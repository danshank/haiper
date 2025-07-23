# Real-time Decision Control: WebSocket vs Server-Sent Events

## Context

We need real-time communication between the web interface and webhook handlers to implement synchronous decision control for Claude Code hooks. Here's a detailed analysis of the approaches:

## Server-Sent Events (SSE)

### How It Works
```go
// Server streams updates to client
func (h *WebHandler) handleTaskEvents(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher := w.(http.Flusher)
    
    // Stream task updates
    for {
        select {
        case update := <-h.taskUpdatesChan:
            fmt.Fprintf(w, "data: %s\n\n", update)
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

### Pros ✅
1. **Simpler Implementation**
   - Built into HTTP standard
   - No special protocol handling
   - Works with standard HTTP/2

2. **Better for Mobile**
   - Lower battery usage
   - More reliable on cellular networks
   - Automatic reconnection in browsers

3. **Firewall Friendly**
   - Uses standard HTTP ports
   - No special proxy configuration
   - Works through corporate firewalls

4. **Perfect for Our Use Case**
   - One-way server→client communication
   - Task status updates are naturally event-driven
   - Built-in reconnection

5. **Lower Resource Usage**
   - No connection upgrade overhead
   - Simpler server-side state management
   - Less memory per connection

6. **Better Error Handling**
   - Automatic retry with exponential backoff
   - Graceful degradation to polling
   - Clear connection state

### Cons ❌
1. **One-way Communication**
   - Client can't send data over SSE connection
   - Need separate HTTP POST for user actions

2. **Browser Connection Limits**
   - HTTP/1.1: ~6 connections per domain
   - Less of an issue with HTTP/2

3. **Proxy Buffering**
   - Some proxies buffer SSE streams
   - Can delay real-time updates

## WebSockets

### How It Works
```go
// Bidirectional connection upgrade
func (h *WebHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    // Handle bidirectional messages
    for {
        var msg TaskAction
        if err := conn.ReadJSON(&msg); err != nil {
            break
        }
        
        // Process user action
        h.handleTaskAction(msg)
        
        // Send response
        conn.WriteJSON(TaskUpdate{Status: "processed"})
    }
}
```

### Pros ✅
1. **Bidirectional Communication**
   - Single connection for both directions
   - Lower latency for interactions
   - Real-time ping/pong

2. **Lower Latency**
   - No HTTP request overhead
   - Direct binary/text frames
   - Instant message delivery

3. **Efficient Protocol**
   - Smaller frame headers
   - Better for high-frequency updates
   - Binary data support

4. **Full Duplex**
   - Simultaneous send/receive
   - Connection state awareness
   - Custom protocols possible

### Cons ❌
1. **Complex Implementation**
   - Connection lifecycle management
   - Custom reconnection logic
   - State synchronization on reconnect

2. **Mobile/Network Issues**
   - Connections drop on network changes
   - Higher battery usage
   - Cellular network challenges

3. **Proxy/Firewall Problems**
   - Some corporate firewalls block WS
   - Proxy servers may not support upgrades
   - Load balancer complexity

4. **Overkill for Our Use Case**
   - We don't need bidirectional streams
   - User actions are infrequent
   - HTTP POST works fine for actions

5. **State Management Complexity**
   - Need to handle connection drops
   - Synchronize state on reconnect
   - Memory leaks with dangling connections

## Our Specific Use Case Analysis

### Communication Pattern
```
1. User opens task page
2. Server streams task updates via SSE
3. User clicks approve/reject (HTTP POST)
4. Server updates task status
5. Webhook handler receives decision
6. SSE streams success confirmation
```

### Traffic Characteristics
- **Low frequency updates** (tasks created every few minutes)
- **One-way heavy** (server→client updates)
- **Infrequent user actions** (approve/reject clicks)
- **Mobile-first usage** (phone notifications)
- **Network reliability** (VPN, cellular)

## Recommendation: Server-Sent Events

### Why SSE is Perfect for This Use Case

1. **Mobile-Optimized**
   ```javascript
   // Automatic reconnection, battery efficient
   const eventSource = new EventSource('/api/task-events');
   eventSource.onmessage = (event) => {
       updateTaskStatus(JSON.parse(event.data));
   };
   ```

2. **Simple Implementation**
   ```go
   // Single channel, easy to understand
   type TaskEventStream struct {
       clients map[string]chan TaskUpdate
   }
   
   func (s *TaskEventStream) NotifyTaskUpdate(taskID string, update TaskUpdate) {
       for _, client := range s.clients {
           select {
           case client <- update:
           default: // Client disconnected
           }
       }
   }
   ```

3. **Hybrid Approach**
   - SSE for real-time updates
   - HTTP POST for user actions
   - Best of both worlds

4. **Production Ready**
   - Works through CDNs
   - Supported by load balancers
   - No special infrastructure

## Implementation Architecture

### SSE + HTTP Hybrid
```
┌─────────────────┐    SSE     ┌──────────────────┐
│   Web Browser   │◄───────────│   Go Server      │
│                 │            │                  │
│  Task Updates   │   HTTP     │  Webhook Handler │
│  ↓              │   POST     │  waiting for     │
│  [ApproveBtn]   │───────────►│  decision        │
└─────────────────┘            └──────────────────┘
```

### Benefits for Our System
1. **Real-time task updates** without polling
2. **Instant decision feedback** to webhook handlers
3. **Mobile-friendly** connection management
4. **Simple deployment** (no special config)
5. **Reliable over VPN** connections

## Final Verdict: Server-Sent Events

SSE is the clear winner for our Claude Code remote control system because:

- ✅ **Perfect fit** for our communication pattern
- ✅ **Mobile-optimized** for phone usage
- ✅ **Simpler** to implement and maintain
- ✅ **More reliable** over networks/VPN
- ✅ **Production-ready** without special infrastructure

WebSockets would be overkill and add unnecessary complexity for our use case.