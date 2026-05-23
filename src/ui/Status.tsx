import { Box, Text } from "ink";
import type React from "react";

export type StatusEmoji = "✅" | "❌" | "📚" | "💾" | "🚀" | "🎉" | "⚠️" | "🔑";

export interface StatusProps {
  emoji?: StatusEmoji;
  color?: string;
  children: React.ReactNode;
}

export function Status({ emoji, color, children }: StatusProps): React.JSX.Element {
  return (
    <Box>
      {emoji ? <Text>{emoji} </Text> : null}
      <Text color={color}>{children}</Text>
    </Box>
  );
}
