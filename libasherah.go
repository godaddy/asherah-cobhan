package main

import (
	"C"
)
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/godaddy/cobhan-go"

	"unsafe"

	"github.com/godaddy/asherah-cobhan/internal/asherah"
	log "github.com/godaddy/asherah-cobhan/internal/log"
	"github.com/godaddy/asherah/go/appencryption"
)

var EstimatedIntermediateKeyOverhead = 0

func main() {
}

//export Shutdown
func Shutdown() {
	log.DebugLog("Asherah shutdown")

	asherah.Shutdown()
}

type Env map[string]string

/*
  Work-around to load environment variables needed by Asherah dependencies,
  because sometimes Go os.Getenv() doesn't see variables set by C.setenv().
  References:
    https://github.com/golang/go/wiki/cgo#environmental-variables
    https://github.com/golang/go/issues/27693
*/
//export SetEnv
func SetEnv(envJson unsafe.Pointer) int32 {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLogf("SetEnv: Panic: %v", r)
			panic(r)
		}
	}()

	cobhan.AllowTempFileBuffers(false)
	env := Env{}

	result := cobhan.BufferToJsonStruct(envJson, &env)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Failed to deserialize environment JSON string %v", cobhan.CobhanErrorToString(result))
		return result
	}

	for k, v := range env {
		os.Setenv(k, v)
	}

	return cobhan.ERR_NONE
}

//export SetupJson
func SetupJson(configJson unsafe.Pointer) int32 {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLogf("SetupJson: Panic: %v", r)
			panic(r)
		}
	}()

	cobhan.AllowTempFileBuffers(false)
	options := &asherah.Options{}
	result := cobhan.BufferToJsonStruct(configJson, options)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Failed to deserialize configuration string %v", cobhan.CobhanErrorToString(result))
		configString, stringResult := cobhan.BufferToString(configJson)
		if stringResult != cobhan.ERR_NONE {
			log.ErrorLogf("Could not convert configJson to string: %v", cobhan.CobhanErrorToString(result))
			return result
		}
		log.ErrorLogf("Could not deserialize: %v", configString)
		return result
	}

	log.EnableVerboseLog(options.Verbose)

	log.DebugLog("Successfully deserialized config JSON")

	EstimatedIntermediateKeyOverhead = len(options.ProductID) + len(options.ServiceName)

	err := asherah.Setup(options)
	if err == asherah.ErrAsherahAlreadyInitialized {
		log.ErrorLog("Setup failed: asherah is already initialized")
		log.ErrorLogf("Setup: asherah.Setup returned %v", err)
		return ERR_ALREADY_INITIALIZED
	}
	if err != nil {
		log.ErrorLog("Setup failed due to bad config?")
		log.ErrorLogf("Setup: asherah.Setup returned %v", err)
		return ERR_BAD_CONFIG
	}

	log.DebugLog("Successfully configured asherah")

	return cobhan.ERR_NONE
}

//export EstimateBuffer
func EstimateBuffer(dataLen int32, partitionLen int32) int32 {
	estimatedDataLen := float64(dataLen+EstimatedEncryptionOverhead) * Base64Overhead
	result := int32(cobhan.BUFFER_HEADER_SIZE + EstimatedEnvelopeOverhead + EstimatedIntermediateKeyOverhead + int(partitionLen) + int(estimatedDataLen))
	return result
}

func EstimateBufferInt(dataLen int, partitionLen int) int {
	return int(EstimateBuffer(int32(dataLen), int32(partitionLen)))
}

//export Decrypt
func Decrypt(partitionIdPtr unsafe.Pointer, encryptedDataPtr unsafe.Pointer, encryptedKeyPtr unsafe.Pointer,
	created int64, parentKeyIdPtr unsafe.Pointer, parentKeyCreated int64, outputDecryptedDataPtr unsafe.Pointer) int32 {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLogf("Decrypt: Panic: %v", r)
			panic(r)
		}
	}()

	encryptedData, result := cobhan.BufferToBytes(encryptedDataPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Decrypt failed: Failed to convert encryptedDataPtr cobhan buffer to bytes %v", cobhan.CobhanErrorToString(result))
		return result
	}

	encryptedKey, result := cobhan.BufferToBytes(encryptedKeyPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Decrypt failed: Failed to convert encryptedKeyPtr cobhan buffer to bytes %v", cobhan.CobhanErrorToString(result))
		return result
	}

	parentKeyId, result := cobhan.BufferToString(parentKeyIdPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Decrypt failed: Failed to convert parentKeyIdPtr cobhan buffer to string %v", cobhan.CobhanErrorToString(result))
		return result
	}

	drr := appencryption.DataRowRecord{
		Data: encryptedData,
		Key: &appencryption.EnvelopeKeyRecord{
			EncryptedKey: encryptedKey,
			Created:      created,
			ParentKeyMeta: &appencryption.KeyMeta{
				ID:      parentKeyId,
				Created: parentKeyCreated,
			},
		},
	}

	data, result, err := decryptData(partitionIdPtr, &drr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Failed to decrypt data %v", cobhan.CobhanErrorToString(result))
		log.ErrorLogf("Decrypt: decryptData returned %v", err)
		return result
	}

	return cobhan.BytesToBuffer(data, outputDecryptedDataPtr)
}

//export Encrypt
func Encrypt(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, outputEncryptedDataPtr unsafe.Pointer,
	outputEncryptedKeyPtr unsafe.Pointer, outputCreatedPtr unsafe.Pointer, outputParentKeyIdPtr unsafe.Pointer,
	outputParentKeyCreatedPtr unsafe.Pointer) int32 {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLogf("Encrypt: Panic: %v", r)
			panic(r)
		}
	}()

	drr, result, err := encryptData(partitionIdPtr, dataPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Failed to encrypt data %v", cobhan.CobhanErrorToString(result))
		log.ErrorLogf("Encrypt failed: encryptData returned %v", err)
		return result
	}

	result = cobhan.BytesToBuffer(drr.Data, outputEncryptedDataPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Encrypted data length: %v", len(drr.Data))
		log.ErrorLogf("Encrypt failed: BytesToBuffer returned %v for outputEncryptedDataPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	result = cobhan.BytesToBuffer(drr.Key.EncryptedKey, outputEncryptedKeyPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Encrypt failed: BytesToBuffer returned %v for outputEncryptedKeyPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.Created, outputCreatedPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Encrypt failed: Int64ToBuffer returned %v for outputCreatedPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	result = cobhan.StringToBuffer(drr.Key.ParentKeyMeta.ID, outputParentKeyIdPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Encrypt failed: BytesToBuffer returned %v for outputParentKeyIdPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.ParentKeyMeta.Created, outputParentKeyCreatedPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Encrypt failed: BytesToBuffer returned %v for outputParentKeyCreatedPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	return cobhan.ERR_NONE
}

//export EncryptToJson
func EncryptToJson(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, jsonPtr unsafe.Pointer) int32 {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLogf("EncryptToJson: Panic: %v", r)
			panic(r)
		}
	}()

	drr, result, err := encryptData(partitionIdPtr, dataPtr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Failed to encrypt data %v", cobhan.CobhanErrorToString(result))
		log.ErrorLogf("EncryptToJson failed: encryptData returned %v", err)
		return result
	}

	result = cobhan.JsonToBuffer(drr, jsonPtr)
	if result != cobhan.ERR_NONE {
		if result == cobhan.ERR_BUFFER_TOO_SMALL {
			outputBytes, err := json.Marshal(drr)
			if err == nil {
				log.ErrorLogf("EncryptToJson failed: JsonToBuffer: Output buffer needed %v bytes", len(outputBytes))
				return result
			}
		}
		log.ErrorLogf("EncryptToJson failed: JsonToBuffer returned %v for jsonPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	return cobhan.ERR_NONE
}

//export DecryptFromJson
func DecryptFromJson(partitionIdPtr unsafe.Pointer, jsonPtr unsafe.Pointer, dataPtr unsafe.Pointer) int32 {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLogf("DecryptFromJson: Panic: %v", r)
			panic(r)
		}
	}()

	var drr appencryption.DataRowRecord
	result := cobhan.BufferToJsonStruct(jsonPtr, &drr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("DecryptFromJson failed: Failed to convert cobhan buffer to JSON structs %v", cobhan.CobhanErrorToString(result))
		return result
	}

	data, result, err := decryptData(partitionIdPtr, &drr)
	if result != cobhan.ERR_NONE {
		log.ErrorLogf("Failed to decrypt data %v", cobhan.CobhanErrorToString(result))
		log.ErrorLogf("DecryptFromJson failed: decryptData returned %v", err)
		return result
	}

	result = cobhan.BytesToBuffer(data, dataPtr)
	if result != cobhan.ERR_NONE {
		if result == cobhan.ERR_BUFFER_TOO_SMALL {
			log.ErrorLogf("DecryptFromJson: BytesToBuffer: Output buffer needed %v bytes", len(data))
			return result
		}
		log.ErrorLogf("DecryptFromJson failed: BytesToBuffer returned %v for dataPtr", cobhan.CobhanErrorToString(result))
		return result
	}

	return cobhan.ERR_NONE
}

func encryptData(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer) (*appencryption.DataRowRecord, int32, error) {
	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != cobhan.ERR_NONE {
		errorMessage := fmt.Sprintf("encryptData failed: Failed to convert cobhan buffer to string %v", cobhan.CobhanErrorToString(result))
		return nil, result, errors.New(errorMessage)
	}

	data, result := cobhan.BufferToBytes(dataPtr)
	if result != cobhan.ERR_NONE {
		errorMessage := fmt.Sprintf("encryptData failed: Failed to convert cobhan buffer to bytes %v", cobhan.CobhanErrorToString(result))
		return nil, result, errors.New(errorMessage)
	}

	drr, err := asherah.Encrypt(partitionId, data)
	if err != nil {
		if err == asherah.ErrAsherahNotInitialized {
			return nil, ERR_NOT_INITIALIZED, err
		}
		return nil, ERR_ENCRYPT_FAILED, err
	}

	return drr, cobhan.ERR_NONE, nil
}

func decryptData(partitionIdPtr unsafe.Pointer, drr *appencryption.DataRowRecord) ([]byte, int32, error) {
	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != cobhan.ERR_NONE {
		errorMessage := fmt.Sprintf("decryptData failed: Failed to convert cobhan buffer to string %v", cobhan.CobhanErrorToString(result))
		log.ErrorLog(errorMessage)
		return nil, result, errors.New(errorMessage)
	}

	data, err := asherah.Decrypt(partitionId, drr)
	if err != nil {
		if err == asherah.ErrAsherahNotInitialized {
			return nil, ERR_NOT_INITIALIZED, err
		}
		return nil, ERR_DECRYPT_FAILED, err
	}

	return data, cobhan.ERR_NONE, nil
}
