//go:build js && wasm

package integration

import (
	"encoding/base64"
	"errors"
	"fmt"
	"syscall/js"
)

const webSaveDataKeyPrefix = "aram.savedata."

var errWebSaveStorageUnavailable = errors.New("browser localStorage is unavailable")

func webSaveStorage() (js.Value, error) {
	global := js.Global()
	if !global.Truthy() {
		return js.Value{}, errWebSaveStorageUnavailable
	}
	storage := global.Get("localStorage")
	if !storage.Truthy() {
		return js.Value{}, errWebSaveStorageUnavailable
	}
	return storage, nil
}

func webSaveDataKey(hash string) (string, error) {
	if hash == "" {
		return "", errors.New("loaded input has no SHA-256 identity")
	}
	return webSaveDataKeyPrefix + hash, nil
}

func (backend *Backend) readSaveData(hash string) (data []byte, err error) {
	key, err := webSaveDataKey(hash)
	if err != nil {
		return nil, err
	}
	// Browser privacy settings may deny localStorage access. Treat that like a
	// first launch so a title can still open without persistent saves.
	defer func() {
		if recover() != nil {
			data, err = nil, nil
		}
	}()
	storage, err := webSaveStorage()
	if err != nil {
		return nil, nil
	}
	value := storage.Call("getItem", key)
	if !value.Truthy() {
		return nil, nil
	}
	data, err = base64.StdEncoding.DecodeString(value.String())
	if err != nil {
		return nil, fmt.Errorf("read game save data: decode browser data: %w", err)
	}
	return data, nil
}

func (backend *Backend) writeSaveDataBlob(hash string, data []byte) (err error) {
	key, err := webSaveDataKey(hash)
	if err != nil {
		return err
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("write browser save data: %v", recovered)
		}
	}()
	storage, err := webSaveStorage()
	if err != nil {
		return err
	}
	storage.Call("setItem", key, base64.StdEncoding.EncodeToString(data))
	return nil
}
