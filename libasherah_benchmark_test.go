package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/godaddy/asherah-cobhan/internal/asherah"
	"github.com/godaddy/cobhan-go"
)

func setupAsherahForBenchmark(b *testing.B, verbose bool) {
	config := &asherah.Options{}

	config.KMS = "static"
	config.ServiceName = "TestService"
	config.ProductID = "TestProduct"
	config.Metastore = "memory"
	config.EnableSessionCaching = true
	config.SessionCacheDuration = time.Hour * 24
	config.SessionCacheMaxSize = 20000
	config.ExpireAfter = time.Hour * 24
	config.CheckInterval = time.Hour * 24
	config.Verbose = verbose
	config.RegionMap = asherah.RegionMap{}
	config.RegionMap["region1"] = "arn1"
	config.RegionMap["region2"] = "arn2"

	buf := benchmarkAllocateJsonBuffer(b, config)

	result := SetupJson(cobhan.Ptr(&buf))
	if result != cobhan.ERR_NONE {
		b.Errorf("SetupJson returned %v", result)
	}
}

func benchmarkAllocateJsonBuffer(b *testing.B, obj interface{}) []byte {
	bytes, err := json.Marshal(obj)
	if err != nil {
		b.Errorf("json.Marshal returned %v", err)
		b.FailNow()
	}
	buf, result := cobhan.AllocateBytesBuffer(bytes)
	if result != cobhan.ERR_NONE {
		b.Error("AllocateBytesBuffer failed")
		b.FailNow()
	}
	return buf
}

func Benchmark_EncryptDecryptRoundTrip(b *testing.B) {
	setupAsherahForBenchmark(b, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		iteration := fmt.Sprint(i)
		cycleEncryptToJsonAndDecryptFromJsonBenchmark("InputString"+iteration, "Partition", b)
	}

	b.StopTimer()

	Shutdown()
}

func cycleEncryptToJsonAndDecryptFromJsonBenchmark(input string, partition string, b *testing.B) {
	partitionIdBuf, result := cobhan.AllocateStringBuffer(partition)
	if result != cobhan.ERR_NONE {
		b.Errorf("AllocateStringBuffer returned %v", result)
		b.FailNow()
	}

	inputBuf, result := cobhan.AllocateStringBuffer(input)
	if result != cobhan.ERR_NONE {
		b.Errorf("AllocateStringBuffer returned %v", result)
		b.FailNow()
	}

	estimatedBufferSize := EstimateBufferInt(len(input), len(partition))
	encryptedDataBuf := cobhan.AllocateBuffer(estimatedBufferSize)

	result = EncryptToJson(cobhan.Ptr(&partitionIdBuf), cobhan.Ptr(&inputBuf), cobhan.Ptr(&encryptedDataBuf))
	if result != cobhan.ERR_NONE {
		b.Errorf("EncryptToJson returned %v", result)
		b.FailNow()
	}

	encrypted_data, result := cobhan.BufferToString(cobhan.Ptr(&encryptedDataBuf))
	if result != cobhan.ERR_NONE {
		b.Errorf("BufferToString returned %v", result)
		b.FailNow()
	}

	encryptedDataInputBuf, result := cobhan.AllocateStringBuffer(encrypted_data)
	if result != cobhan.ERR_NONE {
		b.Errorf("AllocateStringBuffer returned %v", result)
		b.FailNow()
	}

	decryptedDataBuf := cobhan.AllocateBuffer(len(encrypted_data))
	result = DecryptFromJson(cobhan.Ptr(&partitionIdBuf), cobhan.Ptr(&encryptedDataInputBuf), cobhan.Ptr(&decryptedDataBuf))
	if result != cobhan.ERR_NONE {
		b.Errorf("DecryptFromJson returned %v", result)
		b.FailNow()
	}

	decryptedData, result := cobhan.BufferToString(cobhan.Ptr(&decryptedDataBuf))
	if result != cobhan.ERR_NONE {
		b.Errorf("BufferToString returned %v", result)
		b.FailNow()
	}

	if decryptedData != input {
		b.Errorf("decryptedData %v does not match inputData data %v", decryptedData, input)
		b.FailNow()
	}
}
