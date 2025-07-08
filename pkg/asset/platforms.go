//go:generate deno run --allow-read --allow-write ../../schema/gen-platform-constants.ts
//go:generate go fmt platforms_generated.go

package asset