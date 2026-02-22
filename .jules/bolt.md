## 2026-02-21 - Async Metrics Persistence
**Learning:** The application was performing synchronous file I/O on every tool execution, channel send, and memory recall. This was a significant bottleneck and caused race conditions where concurrent operations could overwrite the metrics file.
**Action:** Implemented an asynchronous buffered persistence mechanism for `RuntimeMetrics`. It now aggregates changes in memory and flushes to disk periodically (every 5s) or on shutdown. Future metrics or logging systems should follow this pattern to avoid blocking the main execution loop.
