//go:build js && wasm

package integration

import (
	"bytes"
	"syscall/js"
	"testing"
)

func TestWebSaveDataRoundTrip(t *testing.T) {
	global := js.Global()
	original := global.Get("localStorage")
	storage := global.Get("Object").New()
	values := make(map[string]string)
	getItem := js.FuncOf(func(_ js.Value, args []js.Value) any {
		value, ok := values[args[0].String()]
		if !ok {
			return nil
		}
		return value
	})
	setItem := js.FuncOf(func(_ js.Value, args []js.Value) any {
		values[args[0].String()] = args[1].String()
		return nil
	})
	storage.Set("getItem", getItem)
	storage.Set("setItem", setItem)
	global.Set("localStorage", storage)
	t.Cleanup(func() {
		global.Set("localStorage", original)
		getItem.Release()
		setItem.Release()
	})

	backend := &Backend{}
	want := []byte{0, 1, 2, 0xff, 0x80, 0x00}
	if err := backend.writeSaveData("abc123", want); err != nil {
		t.Fatalf("writeSaveData: %v", err)
	}
	got, err := backend.readSaveData("abc123")
	if err != nil {
		t.Fatalf("readSaveData: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("saved data = %v, want %v", got, want)
	}
}

func TestWebSaveDataReadWithoutLocalStorage(t *testing.T) {
	global := js.Global()
	original := global.Get("localStorage")
	global.Set("localStorage", js.Undefined())
	t.Cleanup(func() { global.Set("localStorage", original) })

	data, err := (&Backend{}).readSaveData("abc123")
	if err != nil {
		t.Fatalf("readSaveData: %v", err)
	}
	if data != nil {
		t.Fatalf("saved data = %v, want nil", data)
	}
}
