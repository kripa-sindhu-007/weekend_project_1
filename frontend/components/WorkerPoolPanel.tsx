"use client";

import { useEffect, useState, useRef } from "react";
import { motion } from "framer-motion";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { usePolling } from "@/lib/hooks";
import { getWorkers } from "@/lib/api";
import { WorkerState } from "@/lib/types";
import { Cpu } from "lucide-react";

function ElapsedTimer({ startedAt }: { startedAt?: string }) {
  const [elapsed, setElapsed] = useState(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (!startedAt) {
      setElapsed(0);
      return;
    }
    const start = new Date(startedAt).getTime();
    const tick = () => setElapsed(Math.floor((Date.now() - start) / 1000));
    tick();
    intervalRef.current = setInterval(tick, 1000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [startedAt]);

  if (!startedAt) return null;
  return <span className="text-xs text-muted-foreground tabular-nums">{elapsed}s</span>;
}

function WorkerCard({ worker }: { worker: WorkerState }) {
  const isProcessing = worker.status === "processing";

  return (
    <motion.div
      layout
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      className={`rounded-lg border p-3 ${
        isProcessing
          ? "border-blue-500/50 bg-blue-500/5 worker-pulse"
          : "border-border bg-card"
      }`}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Cpu className={`w-4 h-4 ${isProcessing ? "text-blue-400" : "text-muted-foreground"}`} />
          <span className="text-sm font-medium">W{worker.id}</span>
        </div>
        <Badge variant={isProcessing ? "info" : "secondary"} className="text-[10px]">
          {worker.status}
        </Badge>
      </div>
      {isProcessing && worker.task_id && (
        <div className="mt-2 flex items-center justify-between">
          <code className="text-xs text-blue-300 truncate max-w-[120px]">{worker.task_id}</code>
          <ElapsedTimer startedAt={worker.started_at} />
        </div>
      )}
    </motion.div>
  );
}

export default function WorkerPoolPanel() {
  const { data: workers } = usePolling(getWorkers, 1000);

  if (!workers) return null;

  const sorted = [...workers].sort((a, b) => a.id - b.id);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Worker Pool</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-2">
          {sorted.map((w) => (
            <WorkerCard key={w.id} worker={w} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
