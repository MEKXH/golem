package session

import (
	"fmt"
	"testing"
)

func BenchmarkSession_Save(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Messages_%d", n), func(b *testing.B) {
			dir := b.TempDir()
			mgr := NewManager(dir)
			sess := mgr.GetOrCreate("bench-save")

			// Pre-fill session
			for i := 0; i < n; i++ {
				sess.AddMessage("user", "test message content")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// We are benchmarking the Save operation which rewrites the file
				if err := mgr.Save(sess); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSession_Append(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Messages_%d", n), func(b *testing.B) {
			dir := b.TempDir()
			mgr := NewManager(dir)
			sess := mgr.GetOrCreate("bench-append")

			// Pre-fill session
			for i := 0; i < n; i++ {
				sess.AddMessage("user", "test message content")
			}
			// Initial save
			mgr.Save(sess)

			newMsg := &Message{Role: "user", Content: "new message"}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := mgr.Append(sess.Key, newMsg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
