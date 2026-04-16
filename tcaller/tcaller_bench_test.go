package tcaller

import "testing"

func BenchmarkCurrent(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Current()
	}
}

func BenchmarkFirstMetadataResolution(b *testing.B) {
	b.ReportAllocs()
	c := helperCurrent()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetCache()
		_ = c.File()
		_ = c.Line()
		_ = c.Function()
	}
}

func BenchmarkCachedMetadataResolution(b *testing.B) {
	b.ReportAllocs()
	resetCache()
	c := helperCurrent()
	_ = c.File()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.File()
		_ = c.Line()
		_ = c.Function()
	}
}

func BenchmarkShortFunction(b *testing.B) {
	b.ReportAllocs()
	resetCache()
	c := (&testService{}).pointerMethodCaller()
	_ = c.ShortFunction()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.ShortFunction()
	}
}
