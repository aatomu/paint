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

type Packet = PacketMouse | PacketAdd | PacketRemove | PacketClear;

type PacketMouse = {
  timestamp: string;
  user: string;
  operation: "mouse";
  data: {
    pos: [number, number];
  };
};

type PacketObjectAdd = {
  timestamp: string;
  user: string;
  operation: "add";
  data: objectPen | objectLine | objectStamp;
};

type objectPen = {
  type: "pen";
  id: string;
  bold: number;
  color: string;
  opacity: number;
  d: string;
};

type objectLine = {
  type: "line";
  id: string;
  bold: number;
  color: string;
  opacity: number;
  start: [number, number];
  end: [number, number];
};

type objectStamp = {
  type: "stamp";
  id: string;
  bold: number;
  color: string;
  opacity: number;
  pos: [number, number];
  text: string[];
};

type PacketRemove = {
  timestamp: string;
  user: string;
  operation: "remove";
  data: {
    target: string;
  };
};

type PacketClear = {
  timestamp: string;
  user: string;
  operation: "clear";
  data: null;
};

