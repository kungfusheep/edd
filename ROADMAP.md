# EDD Project Roadmap


## Feature Ideas

export to / plant / mermaid / text

decision tree diagram graph

more deterministic layouts 

quicky annotate the diagram connections - enter in e mode takes you to the next one down in edit mode - 




### 8. Advanced Features
- **Subgraphs/Clusters**: Group related nodes visually
- **Layers**: Support for multi-layer diagrams
- **Export Formats**: 
  - SVG export
  - PNG export (via terminal screenshot)
  - Mermaid.js conversion
  - GraphViz DOT format
- **Import Formats**:
  - GraphViz DOT files
  - Mermaid.js diagrams
  - PlantUML diagrams

### 9. Developer Experience
- Language Server Protocol (LSP) for JSON diagram files
- Diagram validation and linting
- Auto-layout suggestions
- Refactoring tools (rename nodes, extract subgraph)

## Known Issues to Address
- Bidirectional connections overlap and can be hard to follow
- Very dense diagrams can have overlapping labels
- Large diagrams can be slow to render
- No way to specify preferred routing (e.g., "avoid crossing")

## Future Vision
EDD could become the go-to tool for creating technical diagrams in the terminal, similar to how `tree` is used for directory structures. The focus should remain on simplicity, clarity, and integration with existing terminal workflows.


Example 1

```plantuml
@startuml
skinparam backgroundColor white
skinparam shadowing false

participant "User Browser" as P0
participant "Web Server" as P1
participant "API Gateway" as P2 #51CF66
participant "Auth Service" as P3
participant "Database" as P4 #FF6B6B

P0 -> P1 : GET /login
P1 -> P0 : 200 OK (login page)
P0 -> P1 : POST /login
P1 -> P2 : SOME TEXT
P2 -> P3 : Validate credentials
P3 -> P4 : Query user
P4 --> P3 : User data
P3 -> P3 : Generate JWT
P3 -> P2 : Auth token
P2 -> P1 : Auth success + token
P1 -> P0 : 302 Redirect + Cookie
P0 -> P1 : GET /dashboard
P1 -> P2 : Verify token
P2 -> P3 : Validate JWT
P3 -> P2 : Token valid
P2 -> P4 : Fetch user data
P4 -> P2 : User profile
P2 -> P1 : Dashboard data
P1 -> P0 : 200 OK (dashboard)
P0 -> P1 : POST /logout?
P1 -> P2 : Invalidate session
P2 -> P3 : Revoke token
P3 --> P2 : Token revoked
P2 --> P1 : Logout success
P1 --> P0 : 302 Redirect to /login
@enduml

```

Example 2

```mermaid
sequenceDiagram
    participant P0 as client
    participant P1 as load balancer
    participant P2 as server
    participant P3 as database

    P0->>P1: GET /home
    P1->>P2: /home
    P2->>P3: OFFER DATA
    P3-->>P2: ok
    P2-->>P1: 200 ok
    P1->>P2: **
    P2->>P3: OFFER DATA
    P3-->>P2: ok
    P2-->>P1: ok
    P1-->>P0: 200 ok

```


