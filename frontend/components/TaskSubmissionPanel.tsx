"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { submitTask } from "@/lib/api";
import { Send, Shuffle } from "lucide-react";

export default function TaskSubmissionPanel() {
  const [id, setId] = useState("");
  const [priority, setPriority] = useState(5);
  const [delay, setDelay] = useState(0);
  const [maxRetries, setMaxRetries] = useState(3);
  const [submitting, setSubmitting] = useState(false);
  const [toast, setToast] = useState<{
    msg: string;
    type: "success" | "error";
  } | null>(null);

  const showToast = (msg: string, type: "success" | "error") => {
    setToast({ msg, type });
    setTimeout(() => setToast(null), 3000);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const taskId = id.trim() || `task-${Date.now()}`;
    setSubmitting(true);
    try {
      await submitTask({
        id: taskId,
        priority,
        delay,
        max_retries: maxRetries,
      });
      showToast(`Task "${taskId}" submitted`, "success");
      setId("");
    } catch (err) {
      showToast(
        err instanceof Error ? err.message : "Submit failed",
        "error"
      );
    } finally {
      setSubmitting(false);
    }
  };

  const submitBatch = async () => {
    setSubmitting(true);
    try {
      const promises = Array.from({ length: 10 }, (_, i) =>
        submitTask({
          id: `batch-${Date.now()}-${i}`,
          priority: Math.floor(Math.random() * 10) + 1,
          delay: Math.random() > 0.7 ? Math.floor(Math.random() * 10) + 1 : 0,
          max_retries: 3,
        })
      );
      await Promise.all(promises);
      showToast("10 tasks submitted", "success");
    } catch (err) {
      showToast(
        err instanceof Error ? err.message : "Batch submit failed",
        "error"
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Submit Task</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-xs text-muted-foreground -mt-1">
          Submit a task to see it flow through the pipeline: submit → queue → worker → outcome.
        </p>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="space-y-1">
            <Label htmlFor="task-id">Task ID (auto-generated if empty)</Label>
            <Input
              id="task-id"
              value={id}
              onChange={(e) => setId(e.target.value)}
              placeholder="my-task-1"
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="priority">Priority (1-10, higher = first)</Label>
            <Input
              id="priority"
              type="number"
              min={1}
              max={10}
              value={priority}
              onChange={(e) => setPriority(Number(e.target.value))}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="delay">Delay (seconds)</Label>
            <Input
              id="delay"
              type="number"
              min={0}
              value={delay}
              onChange={(e) => setDelay(Number(e.target.value))}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="retries">Max Retries</Label>
            <Input
              id="retries"
              type="number"
              min={0}
              max={10}
              value={maxRetries}
              onChange={(e) => setMaxRetries(Number(e.target.value))}
            />
          </div>
          <Button className="w-full" disabled={submitting} type="submit">
            <Send className="w-4 h-4" />
            {submitting ? "Submitting..." : "Submit Task"}
          </Button>
        </form>
        <Button
          className="w-full"
          variant="secondary"
          disabled={submitting}
          onClick={submitBatch}
          type="button"
        >
          <Shuffle className="w-4 h-4" />
          Submit 10 Random Tasks
        </Button>

        {toast && (
          <div
            className={`toast rounded-lg px-4 py-2 text-sm border ${
              toast.type === "success"
                ? "bg-emerald-500/10 border-emerald-500 text-emerald-400"
                : "bg-red-500/10 border-red-500 text-red-400"
            }`}
          >
            {toast.msg}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
