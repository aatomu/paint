type BoardConfigration = {
  // UI
  scale: number;
  offset: [number, number];
  // tool
  color: string;
  opacity: number;
  bold: number;
};

type PointerConfiguration = {
  mode: "move" | "pen" | "line" | "stamp" | "delete";
  isDownPrev: boolean;
  isDown: boolean;
  prev: [number, number];
  notify: number;
  current: PacketEventCreate | null;
};

type TransactionConfigration = {
  requestHistory: boolean;
  ws: WebSocket | null;
  heartbeatId: number;
  startTime: number;
  retry: number;
};

// MARK: Packet
// operation:
//   c<=>s :"mouse"|"create"|"delete"|"undo"|"redo"|"clear"
//   s=>c :"success"|"error"
type PacketEvent = {
  packet_id: string;
  user: string;
} & (
  | {
      operation: "";
      data: null;
    }
  | {
      operation: "heatbeat";
      data: PacketEventHeatBeat;
    }
  | {
      operation: "history";
      data: PacketEventHistory;
    }
  | {
      operation: "mouse";
      data: PacketEventMouse;
    }
  | {
      operation: "create";
      data: PacketEventCreate;
    }
  | {
      operation: "delete";
      data: PacketEventDelete;
    }
  | {
      operation: "undo";
      data: PacketEventUndo;
    }
  | {
      operation: "redo";
      data: PacketEventRedo;
    }
  | {
      operation: "clear";
      data: PacketEventClear;
    }
  | {
      operation: "success";
      data: PacketEventSuccess;
    }
  | {
      operation: "error";
      data: PacketEventError;
    }
);

type PacketEventHeatBeat = {};

type PacketEventHistory = {};

type PacketEventMouse = {
  pos: [number, number];
};

type PacketEventCreate = {
  element_id: number;
  bold: number;
  color: string;
  opacity: number;
} & (
  | {
      element_type: "pen";
      property: PropertyPen;
    }
  | {
      element_type: "line";
      property: PropertyLine;
    }
  | {
      element_type: "stamp";
      property: PropertyStamp;
    }
);

type PacketEventDelete = {
  target: number;
};
type PacketEventUndo = {};

type PacketEventRedo = {};

type PacketEventClear = {};

type PacketEventSuccess = {
  event_id: string;
  packet_id: string;
};

type PacketEventError = {
  packet_id: string;
  message: string;
};

// MARK: Generic
type PropertyPen = {
  d: string;
};
type PropertyLine = {
  start: [number, number];
  end: [number, number];
};

type PropertyStamp = {
  pos: [number, number];
  text: string; // ["aaa","bbb", ...]
};
