"use client";

import { motion } from "framer-motion";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { usePolling } from "@/lib/hooks";
import { getEnhancedMetrics } from "@/lib/api";
import { ArrowRight } from "lucide-react";

function AnimatedNumber({ value }: { value: number }) {
  return (
    <motion.span
      key={value}
      initial={{ y: -10, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ type: "spring", stiffness: 300, damping: 25 }}
      className="tabular-nums"
    >
      {value}
    </motion.span>
  );
}

function FlowStage({
  label,
  count,
  color,
}: {
  label: string;
  count: number;
  color: string;
}) {
  return (
    <div className={`flex flex-col items-center gap-1 rounded-lg border border-border bg-card px-4 py-3 min-w-[100px]`}>
      <span className="text-2xl font-bold" style={{ color }}>
        <AnimatedNumber value={count} />
      </span>
      <span className="text-xs text-muted-foreground whitespace-nowrap">{label}</span>
    </div>
  );
}

function Arrow() {
  return (
    <div className="flex items-center text-muted-foreground">
      <ArrowRight className="w-5 h-5" />
    </div>
  );
}

export default function TaskFlowDiagram() {
  const { data: metrics } = usePolling(getEnhancedMetrics, 3000);

  if (!metrics) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Task Flow Pipeline</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-2 overflow-x-auto pb-2 justify-center flex-wrap sm:flex-nowrap">
          <FlowStage label="Submitted" count={metrics.total_submitted} color="#8b5cf6" />
          <Arrow />
          <FlowStage label="Delayed Queue" count={metrics.delayed_queue_size} color="#f59e0b" />
          <Arrow />
          <FlowStage label="Ready Queue" count={metrics.queue_size} color="#3b82f6" />
          <Arrow />
          <FlowStage label="Workers Active" count={metrics.active_workers} color="#06b6d4" />
          <Arrow />
          <div className="flex flex-col gap-1">
            <Badge variant="success" className="text-xs justify-center">
              Completed: {metrics.total_processed}
            </Badge>
            <Badge variant="warning" className="text-xs justify-center">
              Retried: {metrics.total_retries}
            </Badge>
            <Badge variant="destructive" className="text-xs justify-center">
              Dead Letter: {metrics.dead_letter_size}
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
