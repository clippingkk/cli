import { Box, Text } from "ink";
import Spinner from "ink-spinner";
import type React from "react";

export interface SyncProgressProps {
  totalItems: number;
  totalChunks: number;
  completedChunks: number;
  done: boolean;
  error?: string;
}

export function SyncProgress({
  totalItems,
  totalChunks,
  completedChunks,
  done,
  error,
}: SyncProgressProps): React.JSX.Element {
  if (error) {
    return (
      <Box>
        <Text color="red">❌ Sync failed: {error}</Text>
      </Box>
    );
  }

  if (done) {
    return (
      <Box>
        <Text color="green">
          🎉 Successfully uploaded {totalItems} clippings ({totalChunks}/{totalChunks} chunks)
        </Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      <Box>
        <Text color="cyan">
          <Spinner type="dots" />
        </Text>
        <Text>
          {" "}
          Uploading {totalItems} clippings — chunk {completedChunks}/{totalChunks}
        </Text>
      </Box>
    </Box>
  );
}
