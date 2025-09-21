type BoardConfigration = {
  scale: number;
  offset: [number, number];
  color: string;
  opacity: number;
  bold: number;
};

type PointerConfiguration = {
  mode: "move" | "pen" | "line" | "stamp";
  isDownPrev: boolean;
  isDown: boolean;
  prev: [number, number];
  notify: number;
};

// MARK: Packet
// operation:
//   c<=>s :"mouse"|"create"|"delete"|"undo"|"redo"|"clear"
//   s=>c :"success"|"error"
type PacketEvent = {
  packet_id: string;
  name: string;
} & (
  | {
      operation: "";
      data: null;
    }
  | {
      operation: "mouse";
      data: {
        pos: [number, number];
      };
    }
  | {
      operation: "create";
      data: {
        element_id: string;
        bold: number;
        color: string;
        opacity: number;
      } & (
        | {
            type: "pen";
            property: {
              d: string;
            };
          }
        | {
            type: "line";
            property: {
              start: [number, number];
              end: [number, number];
            };
          }
        | {
            type: "stamp";
            property: {
              pos: [number, number];
              text: string;
            };
          }
      );
    }
  | {
      operation: "delete";
      data: {
        target: string;
      };
    }
  | {
      operation: "undo";
      data: {
        target: string;
      };
    }
  | {
      operation: "redo";
      data: {
        target: string;
      };
    }
  | {
      operation: "clear";
    }
  | {
      operation: "success";
      data: {
        packet_id: string;
      };
    }
  | {
      operation: "error";
      data: {
        packet_id: string;
        message: string;
      };
    }
);

type PacketEventMouse = {
  pos: [number, number];
};

type PacketEventCreate = {
  element_id: string;
  bold: number;
  color: string;
  opacity: number;
} & (
  | {
      type: "pen";
      property: PropertyPen;
    }
  | {
      type: "line";
      property: PropertyLine;
    }
  | {
      type: "stamp";
      property: PropertyStamp;
    }
);

type PacketEventDelete = {
  target: string;
};
type PacketEventUndo = {
  target: string;
};
type PacketEventRedo = {
  target: string;
};
type PacketEventClear = {};

type PacketEventSuccess = {
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
