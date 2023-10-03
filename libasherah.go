package main

import (
	"C"
)
import (
	"encoding/json"
	"os"

	"github.com/godaddy/cobhan-go"

	"unsafe"

	"github.com/godaddy/asherah-cobhan/internal/asherah"
	"github.com/godaddy/asherah-cobhan/internal/output"
	"github.com/godaddy/asherah/go/appencryption"
)

var EstimatedIntermediateKeyOverhead = 0

func main() {
}

//export Shutdown
func Shutdown() {
	output.VerboseOutput("Asherah shutdown")

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
	cobhan.AllowTempFileBuffers(false)
	env := Env{}

	result := cobhan.BufferToJsonStruct(envJson, &env)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to deserialize environment JSON string %v", result)
		return result
	}

	for k, v := range env {
		os.Setenv(k, v)
	}

	return cobhan.ERR_NONE
}

//export SetupJson
func SetupJson(configJson unsafe.Pointer) int32 {
	cobhan.AllowTempFileBuffers(false)
	options := &asherah.Options{}
	result := cobhan.BufferToJsonStruct(configJson, options)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to deserialize configuration string %v", result)
		configString, stringResult := cobhan.BufferToString(configJson)
		if stringResult != cobhan.ERR_NONE {
			output.StderrDebugOutputf("Could not convert configJson to string: %v", stringResult)
			return result
		}
		output.StderrDebugOutputf("Could not deserialize: %v", configString)
		return result
	}

	output.EnableVerboseOutput(options.Verbose)

	output.VerboseOutput("Successfully deserialized config JSON")

	EstimatedIntermediateKeyOverhead = len(options.ProductID) + len(options.ServiceName)

	err := asherah.Setup(options)
	if err == asherah.ErrAsherahAlreadyInitialized {
		output.StderrDebugOutput("Setup failed: asherah is already initialized")
		return ERR_ALREADY_INITIALIZED
	}
	if err != nil {
		output.StderrDebugOutput("Setup failed due to bad config?")
		return ERR_BAD_CONFIG
	}

	output.VerboseOutput("Successfully configured asherah")

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
	encryptedData, result := cobhan.BufferToBytes(encryptedDataPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert encryptedDataPtr cobhan buffer to bytes %v", result)
		return result
	}

	encryptedKey, result := cobhan.BufferToBytes(encryptedKeyPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert encryptedKeyPtr cobhan buffer to bytes %v", result)
		return result
	}

	parentKeyId, result := cobhan.BufferToString(parentKeyIdPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert parentKeyIdPtr cobhan buffer to string %v", result)
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

	data, result := decryptData(partitionIdPtr, &drr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to decrypt data %v", result)
		return result
	}

	return cobhan.BytesToBuffer(data, outputDecryptedDataPtr)
}

//export Encrypt
func Encrypt(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, outputEncryptedDataPtr unsafe.Pointer,
	outputEncryptedKeyPtr unsafe.Pointer, outputCreatedPtr unsafe.Pointer, outputParentKeyIdPtr unsafe.Pointer,
	outputParentKeyCreatedPtr unsafe.Pointer) int32 {

	drr, result := encryptData(partitionIdPtr, dataPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to encrypt data %v", result)
		return result
	}

	result = cobhan.BytesToBuffer(drr.Data, outputEncryptedDataPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Encrypted data length: %v", len(drr.Data))
		output.StderrDebugOutputf("Encrypt: BytesToBuffer returned %v for outputEncryptedDataPtr", result)
		return result
	}

	result = cobhan.BytesToBuffer(drr.Key.EncryptedKey, outputEncryptedKeyPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Encrypt: BytesToBuffer returned %v for outputEncryptedKeyPtr", result)
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.Created, outputCreatedPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Encrypt: Int64ToBuffer returned %v for outputCreatedPtr", result)
		return result
	}

	result = cobhan.StringToBuffer(drr.Key.ParentKeyMeta.ID, outputParentKeyIdPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Encrypt: BytesToBuffer returned %v for outputParentKeyIdPtr", result)
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.ParentKeyMeta.Created, outputParentKeyCreatedPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Encrypt: BytesToBuffer returned %v for outputParentKeyCreatedPtr", result)
		return result
	}

	return cobhan.ERR_NONE
}

//export EncryptToJson
func EncryptToJson(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, jsonPtr unsafe.Pointer) int32 {
	drr, result := encryptData(partitionIdPtr, dataPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to encrypt data %v", result)
		return result
	}

	result = cobhan.JsonToBuffer(drr, jsonPtr)
	if result != cobhan.ERR_NONE {
		if result == cobhan.ERR_BUFFER_TOO_SMALL {
			outputBytes, err := json.Marshal(drr)
			if err == nil {
				output.StderrDebugOutputf("EncryptToJson: JsonToBuffer: Output buffer needed %v bytes", len(outputBytes))
				return result
			}
		}
		output.StderrDebugOutputf("EncryptToJson: JsonToBuffer returned %v for jsonPtr", result)
		return result
	}

	return cobhan.ERR_NONE
}

//export DecryptFromJson
func DecryptFromJson(partitionIdPtr unsafe.Pointer, jsonPtr unsafe.Pointer, dataPtr unsafe.Pointer) int32 {
	var drr appencryption.DataRowRecord
	result := cobhan.BufferToJsonStruct(jsonPtr, &drr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert cobhan buffer to JSON structs %v", result)
		return result
	}

	data, result := decryptData(partitionIdPtr, &drr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to decrypt data %v", result)
		return result
	}

	result = cobhan.BytesToBuffer(data, dataPtr)
	if result != cobhan.ERR_NONE {
		if result == cobhan.ERR_BUFFER_TOO_SMALL {
			output.StderrDebugOutputf("DecryptFromJson: BytesToBuffer: Output buffer needed %v bytes", len(data))
			return result
		}
		output.StderrDebugOutputf("DecryptFromJson: BytesToBuffer returned %v for dataPtr", result)
		return result
	}

	return cobhan.ERR_NONE
}

func encryptData(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer) (*appencryption.DataRowRecord, int32) {
	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert cobhan buffer to string %v", result)
		return nil, result
	}

	data, result := cobhan.BufferToBytes(dataPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert cobhan buffer to bytes %v", result)
		return nil, result
	}

	drr, err := asherah.Encrypt(partitionId, data)
	if err != nil {
		if err == asherah.ErrAsherahNotInitialized {
			output.StderrDebugOutput("Encrypt failed: asherah is not initialized")
			return nil, ERR_NOT_INITIALIZED
		}
		output.StderrDebugOutputf("Encrypt failed: %v", err)
		return nil, ERR_ENCRYPT_FAILED
	}

	return drr, cobhan.ERR_NONE
}

func decryptData(partitionIdPtr unsafe.Pointer, drr *appencryption.DataRowRecord) ([]byte, int32) {
	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != cobhan.ERR_NONE {
		output.StderrDebugOutputf("Failed to convert cobhan buffer to string %v", result)
		return nil, result
	}

	data, err := asherah.Decrypt(partitionId, drr)
	if err != nil {
		if err == asherah.ErrAsherahNotInitialized {
			output.StderrDebugOutput("Decrypt failed: asherah is not initialized")
			return nil, ERR_NOT_INITIALIZED
		}
		output.StderrDebugOutputf("Decrypt failed: %v", err)
		return nil, ERR_DECRYPT_FAILED
	}

	return data, cobhan.ERR_NONE
}
