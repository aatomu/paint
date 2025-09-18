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
  timestamp: number;
  user: string;
} & (
  | {
      operation: "mouse";
      data: {
        posX: number;
        posY: number;
      };
    }
  | {
      operation: "objectAdd";
      data: {
        id: string;
        bold: number;
        color: string;
        opacity: number;
      } & (
        | {
            type: "pen";
            d: string;
          }
        | {
            type: "line";
            startX: number;
            startY: number;
            endX: number;
            endY: number;
          }
        | {
            type: "stamp";
            posX: number;
            posY: number;
            text: string;
          }
      );
    }
  | {
      operation: "objectRemove";
      data: {
        target: string;
      };
    }
  | {
      operation: "boardClear";
      data?: null;
    }
);
