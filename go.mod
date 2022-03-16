module github.com/godaddy/asherah-cobhan

go 1.16

replace github.com/godaddy/asherah-cobhan/internal/asherah_internals => ./internal/asherah_internals

replace github.com/godaddy/asherah-cobhan/internal/debug_output => ./internal/debug_output

require (
	github.com/awnumar/memcall v0.1.2 // indirect
	github.com/awnumar/memguard v0.22.2 // indirect
	github.com/aws/aws-sdk-go v1.43.20 // indirect
	github.com/goburrow/cache v0.1.4 // indirect
	github.com/godaddy/asherah-cobhan/internal/asherah_internals v0.0.0-00010101000000-000000000000
	github.com/godaddy/asherah-cobhan/internal/debug_output v0.0.0-00010101000000-000000000000
	github.com/godaddy/asherah/go/appencryption v0.2.3
	github.com/godaddy/cobhan-go v0.2.0
	golang.org/x/crypto v0.0.0-20220315160706-3147a52a75dd // indirect
	golang.org/x/sys v0.0.0-20220315194320-039c03cc5b86 // indirect
)
