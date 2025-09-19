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
  current: string;
  data: string;
};

type PacketEvent = {
  eventId: number;
} & (
  | {
      operation: "mouse";
      data: {
        pos: [number, number];
      };
    }
  | {
      operation: "create";
      data: {
        id: string;
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
              start: [number,number]
              end: [number,number]
            }
          }
        | {
            type: "stamp";
            pos: [number,number]
            text: string;
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
);
