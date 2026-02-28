## 2026-02-21 - Async Metrics Persistence
**Learning:** The application was performing synchronous file I/O on every tool execution, channel send, and memory recall. This was a significant bottleneck and caused race conditions where concurrent operations could overwrite the metrics file.
**Action:** Implemented an asynchronous buffered persistence mechanism for `RuntimeMetrics`. It now aggregates changes in memory and flushes to disk periodically (every 5s) or on shutdown. Future metrics or logging systems should follow this pattern to avoid blocking the main execution loop.

## 2026-02-21 - Caching Static Context Files
**Learning:** The agent was re-reading 5-8 static configuration files (IDENTITY.md, etc.) and scanning the skills directory on every single interaction turn (LLM request), causing unnecessary disk I/O and latency.
**Action:** Implemented a caching mechanism in ContextBuilder for the base system prompt parts. The cache is invalidated only when file modification tools (write_file, etc.) are executed, ensuring both performance and correctness.

## 2026-02-23 - Selective Cache Invalidation
**Learning:** The previous caching mechanism invalidated the entire system prompt cache whenever *any* file was modified by the agent. This was inefficient for long coding sessions where the agent primarily modifies source code, which doesn't affect the static system prompt.
**Action:** Refined the cache invalidation logic in `ContextBuilder` to check the path of the modified file. The cache is now only cleared if the modified file is one of the base context files (e.g., IDENTITY.md) or resides in the `skills/` directory. This significantly reduces I/O overhead during development tasks.

## 2026-03-09 - Session Manager Lock Contention
**Learning:** The `SessionManager.GetOrCreate` method was acquiring a full write lock (`m.mu.Lock()`) on the entire `Manager` even for retrieving existing sessions. In a multi-channel/server environment where the agent processes messages concurrently, this causes significant lock contention because the vast majority of calls hit existing sessions.
**Action:** Implemented a double-checked locking pattern using a fast-path read lock (`m.mu.RLock()`). The write lock is now only acquired when an actual cache miss occurs (i.e., when creating a new session from disk or memory).
