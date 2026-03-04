"use client";

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { usePolling } from "@/lib/hooks";
import { getEnhancedMetrics } from "@/lib/api";
import {
  Activity,
  CheckCircle,
  XCircle,
  RotateCcw,
  Layers,
  Users,
  Clock,
  Skull,
} from "lucide-react";

function StatCard({
  label,
  value,
  icon: Icon,
  color,
}: {
  label: string;
  value: number | string;
  icon: React.ComponentType<{ className?: string }>;
  color: string;
}) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border bg-background/50 p-3">
      <div className="shrink-0" style={{ color }}>
        <Icon className="w-4 h-4" />
      </div>
      <div className="min-w-0">
        <div className="text-lg font-bold tabular-nums" style={{ color }}>
          {value}
        </div>
        <div className="text-[10px] text-muted-foreground uppercase tracking-wider">
          {label}
        </div>
      </div>
    </div>
  );
}

export default function MetricsPanel() {
  const { data: metrics, error } = usePolling(getEnhancedMetrics, 3000);

  if (error) {
    return (
      <Card>
        <CardHeader><CardTitle>Metrics</CardTitle></CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">{error}</p>
        </CardContent>
      </Card>
    );
  }

  if (!metrics) {
    return (
      <Card>
        <CardHeader><CardTitle>Metrics</CardTitle></CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">Loading...</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader><CardTitle>Metrics</CardTitle></CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-2">
          <StatCard label="Processed" value={metrics.total_processed} icon={CheckCircle} color="#22c55e" />
          <StatCard label="Failed" value={metrics.total_failed} icon={XCircle} color="#ef4444" />
          <StatCard label="Retries" value={metrics.total_retries} icon={RotateCcw} color="#f59e0b" />
          <StatCard label="Queue Size" value={metrics.queue_size} icon={Layers} color="#3b82f6" />
          <StatCard label="Active Workers" value={metrics.active_workers} icon={Users} color="#06b6d4" />
          <StatCard label="Delayed" value={metrics.delayed_queue_size} icon={Clock} color="#f59e0b" />
          <StatCard label="Dead Letter" value={metrics.dead_letter_size} icon={Skull} color="#ef4444" />
          <StatCard label="Submitted" value={metrics.total_submitted} icon={Activity} color="#8b5cf6" />
        </div>

        <div className="space-y-1.5">
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">Success Rate</span>
            <span className="text-xs font-semibold tabular-nums text-emerald-400">
              {metrics.success_rate.toFixed(1)}%
            </span>
          </div>
          <Progress value={metrics.success_rate} className="h-2" />
        </div>
      </CardContent>
    </Card>
  );
}
