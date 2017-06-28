package bungie

import "testing"

// NOTE: Never run this while using the bungie.net URLs in bungie/constants.go
// those should be changed to a localhost webserver that returns static results.
func BenchmarkSomething(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CountItem("strange coins", "aaabbbccc")
	}
}
