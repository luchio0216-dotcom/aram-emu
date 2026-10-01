package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"time"

	"github.com/mirusu400/aram-core/application"
	"github.com/mirusu400/aram-core/core"
	"github.com/mirusu400/aram-core/cpu"
)

type command struct {
	Control string `json:"control"`
	Pressed bool `json:"pressed"`
	Frames int `json:"frames"`
	Label string `json:"label"`
}

func check(err error) { if err != nil { panic(err) } }

func main() {
	game := flag.String("game", "", "authorized game ZIP")
	save := flag.String("savedata", "", "private persistent ARAM save")
	load := flag.String("load-state", "", "private full machine state")
	script := flag.String("script", "", "JSON command array")
	mode := flag.String("cpu", "jit", "CPU backend")
	out := flag.String("out", "", "private output directory")
	budget := flag.Uint64("budget", 0, "override KTF guest instructions per presentation quantum")
	perFrame := flag.Bool("frame-metrics", false, "record guest frame counters for each host quantum")
	warmup := flag.Int("warmup", 0, "warmup host frames before profiling")
	profile := flag.Bool("profile", false, "record CPU profile")
	flag.Parse()
	check(os.MkdirAll(*out, 0700))
	data, err := os.ReadFile(*game); check(err)
	newCPU, err := application.ResolveCPUBackend(*mode); check(err)
	var b cpu.Backend
	f := application.NewFactory()
	f.FrameRunBudget = application.DefaultHandsetRunBudget
	f.KTFRunBudget = application.DefaultKTFHandsetRunBudget
	if *budget != 0 { f.KTFRunBudget = *budget }
	f.NewCPU = func() cpu.Backend { b = newCPU(); return b }
	ctx := context.Background()
	m, err := f.Create(ctx, core.Source{Name: filepath.Base(*game), ReaderAt: bytes.NewReader(data), Size: int64(len(data)), SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}); check(err)
	defer m.Close()
	if *save != "" {
		s, err := os.ReadFile(*save); check(err)
		check(m.(interface { ImportSaveData([]byte) error }).ImportSaveData(s))
	}
	if *load != "" {
		s, err := os.Open(*load); check(err); check(m.LoadState(s)); check(s.Close())
		if m.State() == core.StatePaused { check(m.Resume()) }
	} else { check(m.Start(ctx)) }
	var commands []command
	s, err := os.ReadFile(*script); check(err); check(json.Unmarshal(s, &commands))
	for frame := 0; frame < *warmup; frame++ { check(m.StepFrame(ctx)); for { audio := m.DrainAudio(); if len(audio.PCM16) == 0 { break } } }
	frameMetrics := make([]map[string]any, 0)
	var profileFile *os.File
	if *profile { profileFile, err = os.Create(filepath.Join(*out, "cpu.pprof")); check(err); check(pprof.StartCPUProfile(profileFile)) }
	reports := make([]map[string]any, 0, len(commands))
	for _, c := range commands {
		if c.Control != "" { check(m.QueueInput(core.InputEvent{Control: c.Control, Pressed: c.Pressed})) }
		before := m.(*application.Machine).DebugSnapshot(1)
		durations := make([]int64, 0, c.Frames)
		start := time.Now()
		for frame := 0; frame < c.Frames; frame++ {
			tick := time.Now(); check(m.StepFrame(ctx)); durations = append(durations, time.Since(tick).Microseconds())
			for { audio := m.DrainAudio(); if len(audio.PCM16) == 0 { break } }
			if *perFrame { frameMetrics = append(frameMetrics, map[string]any{"label":c.Label,"frame":frame,"duration_us":durations[len(durations)-1],"snapshot":m.(*application.Machine).DebugSnapshot(0)}) }
		}
		elapsed := time.Since(start)
		after := m.(*application.Machine).DebugSnapshot(1)
		sort.Slice(durations, func(i,j int) bool { return durations[i] < durations[j] })
		q := func(p int) int64 { if len(durations) == 0 { return 0 }; return durations[(len(durations)-1)*p/100] }
		r := map[string]any{"label":c.Label,"frames":c.Frames,"wall_ms":float64(elapsed.Microseconds())/1000,"p50_us":q(50),"p95_us":q(95),"p99_us":q(99),"max_us":q(100),"before":before,"after":after}
		read32 := func(a uint32) uint32 { buf := make([]byte,4); if b.ReadMemory(a,buf) != nil { return 0 }; return binary.LittleEndian.Uint32(buf) }
		ptr := read32(0x2aaa78)
		coords := make([]byte,4)
		if ptr != 0 && b.ReadMemory(ptr+2,coords) == nil { r["player_xy"] = []int16{int16(binary.LittleEndian.Uint16(coords)),int16(binary.LittleEndian.Uint16(coords[2:]))} }
		r["ground_count"] = read32(0x194c6c)
		r["player_ptr"] = ptr
		if c.Label != "" {
			f, err := os.Create(filepath.Join(*out,c.Label+".png")); check(err); check(png.Encode(f,m.Framebuffer())); check(f.Close())
			f, err = os.Create(filepath.Join(*out,c.Label+".state")); check(err); check(m.SaveState(f)); check(f.Close())
			memory := make([]byte,608192+1149832); check(b.ReadMemory(0x100000,memory)); check(os.WriteFile(filepath.Join(*out,c.Label+".bin"),memory,0600))
		}
		reports = append(reports,r)
	}
	if *profile { pprof.StopCPUProfile(); check(profileFile.Close()) }
	encoded, err := json.MarshalIndent(reports,"","  "); check(err); check(os.WriteFile(filepath.Join(*out,"measurements.json"),encoded,0600))
	if *perFrame { encoded, err := json.Marshal(frameMetrics); check(err); check(os.WriteFile(filepath.Join(*out,"frames.json"),encoded,0600)) }
	fmt.Printf("completed %d private scenarios using %s\n",len(commands),*mode)
}
