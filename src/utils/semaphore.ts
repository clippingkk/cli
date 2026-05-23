export async function withConcurrency<T>(
  limit: number,
  tasks: Array<() => Promise<T>>,
): Promise<T[]> {
  if (limit < 1) throw new Error("limit must be >= 1");
  const results: T[] = Array.from({ length: tasks.length });
  let nextIndex = 0;

  async function worker(): Promise<void> {
    while (true) {
      const i = nextIndex++;
      if (i >= tasks.length) return;
      const task = tasks[i];
      if (!task) return;
      results[i] = await task();
    }
  }

  const workers: Promise<void>[] = [];
  const workerCount = Math.min(limit, tasks.length);
  for (let i = 0; i < workerCount; i++) {
    workers.push(worker());
  }
  await Promise.all(workers);
  return results;
}
