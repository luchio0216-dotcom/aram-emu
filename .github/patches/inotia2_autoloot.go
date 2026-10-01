package application

import (
 "bytes"
 "crypto/sha256"
 "encoding/hex"
 "fmt"
)

const inotia2AutoLootOriginalSHA = "e9dc70fa04194aed2e5051a3da92ff6b7999506fd85d69f432b6abba7f5d8551"
const inotia2AutoLootPatchedSHA = "36b0e69210923dee4d622d239dc9ff900f07780e921c860cb6aa9d541b312aff"

type inotia2ImagePatch struct {
 offset int
 expectedSHA string
 replacement []byte
}

func inotia2PatchBytes(value string) []byte {
 data, err := hex.DecodeString(value)
 if err != nil { panic(err) }
 return data
}

// These are newly assembled Thumb helpers, not game assets. The known Rare12
// client already returns from its offline-ready check at 0x12af8c; reclaim
// only that unreachable body and leave the text/BSS boundaries unchanged.
// Empty ground lists skip pickup entirely. The automatic-call flag lives in
// four new BSS bytes, away from JIT-translated code and native store barriers.
// The actor update hook preserves the original actor tag and callee registers,
// then calls the game's proximity pickup only for the currently selected hero.
// Native pickup retains its slot/stack checks and removes drops only after
// insertion succeeds. Its full-inventory message is suppressed only while
// this synchronous automatic call is running; manual OK retains its behavior.
var inotia2AutoLootPatches = []inotia2ImagePatch{
 {offset: 0x37c2a, expectedSHA: "16126326ddc04c123c19ae8a51ad876978108d3e82402d382c07624dafb1a8ce", replacement: inotia2PatchBytes("f3f7b1f9")},
 {offset: 0x2ce38, expectedSHA: "b0392ee1e89ed4f94b06aac8a28859b9f0e6524d6e31dba44b415d7e070a1487", replacement: inotia2PatchBytes("fef7ccf8")},
 {offset: 0x2af90, expectedSHA: "d2732bb9ad6321ccc1ff83b7de06db67497faa81b39420ed4b4439b721ca1aca", replacement: inotia2PatchBytes("70b5044601f0f4fa0546002d0fd0094909688c420bd108490968002907d0074e01233360204601f09bfe00233360284670bd00bf78aa2a006c4c190048d32a00c046c04608b5054b1b68002b02d0044b019308bd01200ba908bd00bf48d32a001bce1200c046c046c046c046c046c046c046c046c046c046c046c046c046c04600000000")},
}

func inotia2AutoLootClient(client []byte) ([]byte, error) {
 digest := fmt.Sprintf("%x", sha256.Sum256(client))
 if digest == inotia2AutoLootPatchedSHA { return client, nil }
 return applyInotia2ImagePatches(client, inotia2AutoLootOriginalSHA, inotia2AutoLootPatches)
}

// Check every expected span before changing a clone. Unknown clients pass
// through unchanged; corrupt or overlapping patch definitions fail atomically.
func applyInotia2ImagePatches(client []byte, imageSHA string, patches []inotia2ImagePatch) ([]byte, error) {
 if fmt.Sprintf("%x", sha256.Sum256(client)) != imageSHA { return client, nil }
 for i,p := range patches {
  if p.offset < 0 || p.offset > len(client) || len(p.replacement) > len(client)-p.offset {
   return nil, fmt.Errorf("Inotia2 automatic pickup patch %d is outside client", i)
  }
  span := client[p.offset:p.offset+len(p.replacement)]
  if fmt.Sprintf("%x", sha256.Sum256(span)) != p.expectedSHA {
   return nil, fmt.Errorf("Inotia2 automatic pickup patch %d original checksum mismatch", i)
  }
  for _,q := range patches[:i] {
   if p.offset < q.offset+len(q.replacement) && q.offset < p.offset+len(p.replacement) {
    return nil, fmt.Errorf("Inotia2 automatic pickup patch %d overlaps an earlier patch", i)
   }
  }
 }
 out := bytes.Clone(client)
 for _,p := range patches { copy(out[p.offset:],p.replacement) }
 return out,nil
}


// Append scratch storage after the validated original BSS. Native globals keep
// their addresses; the original ZIP/save identity is unchanged.
func inotia2AutoLootBSSSize(client []byte, size uint32) (uint32, error) {
 if fmt.Sprintf("%x", sha256.Sum256(client)) != inotia2AutoLootPatchedSHA { return size, nil }
 if size == 1149836 { return size, nil }
 if size != 1149832 { return 0, fmt.Errorf("Inotia2 automatic pickup BSS baseline mismatch: %d", size) }
 return size + 4, nil
}
