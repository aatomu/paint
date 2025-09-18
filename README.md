# paint
Golangで自分のために作成  
開発環境:rpi4(8gb) golang(go1.17.8 linux/arm64)  

## -起動-  
```go run main.go```
  
## アクセス  
`http://<localIP>/?room=<roomID>`
## コード元:  
Bot Language   : https://golang.org/  


- Top:
  - Input:
    - Name: string(user name)
    - Room: string(room name)
  - Button:
    - Submit: jump to Room
- Room:
  - Button:
    - Select Pen
    - Color Selector
    - Undo
    - Redo
    - Clear All
    - Copy Link
    - Text
    - Box
    - Line
    - Circle
  - Other:
    - Zoom in
    - Zoom out
    - Move


rooms.db ER mappings
```mermaid
erDiagram
  event {
    integer event_id PK "auto generate"
    text timestamp "send by client"
    text user "username"
    text operation "objectAdd|objectRemove|boardClear"
  }
  event_object_add {
    integer event_id PK,FK "origin event"
    text id "object id"
    real bold  "object bold"
    color text "object color"
    real opacity "object opacity"
    text type "object type, pen|line|stamp"
  }
  event_object_remove {
    integer event_id FK "orign event"
    text target "target object id"
  }
  object_pen {
    integer id FK "origin object"
    text d "pen path"
  }
  object_line {
    integer id FK "origin object"
    real start_x
    real start_y
    real end_x
    real end_x
  }
  object_stamp {
    integer id FK "origin object"
    real pos_x
    real pos_y
    text text  "stamp text, eg:[#quot;aaa#quot;,#quot;bbb#quot;, ... ]"
  }

  event ||--o{ event_object_add : ""
  event ||--o{ event_object_remove : ""
  event_object_add ||--o{ object_pen : ""
  event_object_add ||--o{ object_line : ""
  event_object_add ||--o{ object_stamp : ""
```