package libasherah

import (
	"C"
)
import (
	"context"
	"encoding/json"

	"github.com/godaddy/cobhan-go"

	"unsafe"

	"github.com/godaddy/asherah/go/appencryption"	
	"github.com/godaddy/asherah-cobhan/internal/asherah_internals"
	"github.com/godaddy/asherah-cobhan/internal/debug_output"
)

const ERR_NONE = 0
const ERR_NOT_INITIALIZED = -100
const ERR_ALREADY_INITIALIZED = -101
const ERR_GET_SESSION_FAILED = -102
const ERR_ENCRYPT_FAILED = -103
const ERR_DECRYPT_FAILED = -104
const ERR_BAD_CONFIG = -105

const EstimatedEncryptionOverhead = 48
const EstimatedEnvelopeOverhead = 185
const Base64Overhead = 1.34

var EstimatedIntermediateKeyOverhead = 0

func main() {
}

var globalDebugOutput func(interface{}) = nil
var globalDebugOutputf func(format string, args ...interface{}) = nil

//export Shutdown
func Shutdown() {
	ShutdownAsherah()
}

//export SetupJson
func SetupJson(configJson unsafe.Pointer) int32 {
	cobhan.AllowTempFileBuffers(false)
	options := &Options{}
	result := cobhan.BufferToJsonStruct(configJson, options)
	if result != ERR_NONE {
		StdoutDebugOutputf("Failed to deserialize configuration string %v", result)
		configString, stringResult := cobhan.BufferToString(configJson)
		if stringResult != ERR_NONE {
			return result
		}
		StdoutDebugOutputf("Could not deserialize: %v", configString)
		return result
	}

	if options.Verbose {
		globalDebugOutput = StdoutDebugOutput
		globalDebugOutputf = StdoutDebugOutputf
		globalDebugOutput("Enabled debug output")
	} else {
		globalDebugOutput = NullDebugOutput
		globalDebugOutputf = NullDebugOutputf
	}

	globalDebugOutput("Successfully deserialized config JSON")
	globalDebugOutput(options)

	EstimatedIntermediateKeyOverhead = len(options.ProductID) + len(options.ServiceName)

	SetupAsherah(options)

	return ERR_NONE
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
	if result != ERR_NONE {
		return result
	}

	encryptedKey, result := cobhan.BufferToBytes(encryptedKeyPtr)
	if result != ERR_NONE {
		return result
	}

	parentKeyId, result := cobhan.BufferToString(parentKeyIdPtr)
	if result != ERR_NONE {
		return result
	}

	globalDebugOutputf("parentKeyId: %v", parentKeyId)

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
	if result != ERR_NONE {
		return result
	}

	return cobhan.BytesToBuffer(data, outputDecryptedDataPtr)
}

//export Encrypt
func Encrypt(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, outputEncryptedDataPtr unsafe.Pointer,
	outputEncryptedKeyPtr unsafe.Pointer, outputCreatedPtr unsafe.Pointer, outputParentKeyIdPtr unsafe.Pointer,
	outputParentKeyCreatedPtr unsafe.Pointer) int32 {

	drr, result := encryptData(partitionIdPtr, dataPtr)
	if result != ERR_NONE {
		return result
	}

	result = cobhan.BytesToBuffer(drr.Data, outputEncryptedDataPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypted data length: %v", len(drr.Data))
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputEncryptedDataPtr", result)
		return result
	}

	result = cobhan.BytesToBuffer(drr.Key.EncryptedKey, outputEncryptedKeyPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputEncryptedKeyPtr", result)
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.Created, outputCreatedPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: Int64ToBuffer returned %v for outputCreatedPtr", result)
		return result
	}

	result = cobhan.StringToBuffer(drr.Key.ParentKeyMeta.ID, outputParentKeyIdPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputParentKeyIdPtr", result)
		return result
	}

	result = cobhan.Int64ToBuffer(drr.Key.ParentKeyMeta.Created, outputParentKeyCreatedPtr)
	if result != ERR_NONE {
		globalDebugOutputf("Encrypt: BytesToBuffer returned %v for outputParentKeyCreatedPtr", result)
		return result
	}

	return ERR_NONE
}

//export EncryptToJson
func EncryptToJson(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer, jsonPtr unsafe.Pointer) int32 {
	drr, result := encryptData(partitionIdPtr, dataPtr)
	if result != ERR_NONE {
		return result
	}

	result = cobhan.JsonToBuffer(drr, jsonPtr)
	if result != ERR_NONE {
		if result == cobhan.ERR_BUFFER_TOO_SMALL {
			outputBytes, err := json.Marshal(drr)
			if err == nil {
				globalDebugOutputf("EncryptToJson: JsonToBuffer: Output buffer needed %v bytes", len(outputBytes))
				return result
			}
		}
		globalDebugOutputf("EncryptToJson: JsonToBuffer returned %v for jsonPtr", result)
		return result
	}

	return ERR_NONE
}

//export DecryptFromJson
func DecryptFromJson(partitionIdPtr unsafe.Pointer, jsonPtr unsafe.Pointer, dataPtr unsafe.Pointer) int32 {
	var drr appencryption.DataRowRecord
	result := cobhan.BufferToJsonStruct(jsonPtr, &drr)
	if result != ERR_NONE {
		return result
	}

	data, result := decryptData(partitionIdPtr, &drr)
	if result != ERR_NONE {
		return result
	}

	result = cobhan.BytesToBuffer(data, dataPtr)
	if result != ERR_NONE {
		if result == cobhan.ERR_BUFFER_TOO_SMALL {
			globalDebugOutputf("DecryptFromJson: BytesToBuffer: Output buffer needed %v bytes", len(data))
			return result
		}
		globalDebugOutputf("DecryptFromJson: BytesToBuffer returned %v for dataPtr", result)
		return result
	}

	return ERR_NONE
}

func encryptData(partitionIdPtr unsafe.Pointer, dataPtr unsafe.Pointer) (*appencryption.DataRowRecord, int32) {
	if globalInitialized == 0 {
		return nil, ERR_NOT_INITIALIZED
	}

	globalDebugOutput("Encrypt()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != ERR_NONE {
		return nil, result
	}

	data, result := cobhan.BufferToBytes(dataPtr)
	if result != ERR_NONE {
		return nil, result
	}

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutputf("Encrypt: GetSession failed: %v", err)
		return nil, ERR_GET_SESSION_FAILED
	}
	defer session.Close()

	ctx := context.Background()
	drr, err := session.Encrypt(ctx, data)
	if err != nil {
		globalDebugOutput("Encrypt failed: " + err.Error())
		return nil, ERR_ENCRYPT_FAILED
	}

	return drr, ERR_NONE
}

func decryptData(partitionIdPtr unsafe.Pointer, drr *appencryption.DataRowRecord) ([]byte, int32) {
	if globalInitialized == 0 {
		return nil, ERR_NOT_INITIALIZED
	}

	globalDebugOutput("Decrypt()")

	partitionId, result := cobhan.BufferToString(partitionIdPtr)
	if result != ERR_NONE {
		return nil, result
	}

	globalDebugOutputf("Decrypting with partition: %v", partitionId)

	session, err := globalSessionFactory.GetSession(partitionId)
	if err != nil {
		globalDebugOutput(err.Error())
		return nil, ERR_GET_SESSION_FAILED
	}
	defer session.Close()

	ctx := context.Background()
	data, err := session.Decrypt(ctx, *drr)
	if err != nil {
		globalDebugOutputf("Decrypt failed: %v", err)
		return nil, ERR_DECRYPT_FAILED
	}

	return data, ERR_NONE
}
