export interface Task {
  id: string;
  priority: number;
  delay: number;
  max_retries: number;
  retries: number;
  status: "pending" | "processing" | "completed" | "failed";
  created_at: string;
  error?: string;
}

export interface Metrics {
  total_processed: number;
  total_failed: number;
  total_retries: number;
  queue_size: number;
  active_workers: number;
}

export interface FailedTask {
  task: Task;
  failed_at: string;
  reason: string;
}

export interface SubmitTaskRequest {
  id: string;
  priority: number;
  delay: number;
  max_retries: number;
}

export interface TaskEvent {
  id: string;
  task_id: string;
  type: "submitted" | "started" | "completed" | "failed" | "retrying" | "dead_lettered" | "promoted";
  worker_id: number;
  detail: string;
  timestamp: string;
}

export interface WorkerState {
  id: number;
  status: "idle" | "processing";
  task_id?: string;
  started_at?: string;
}

export interface DelayedEntry {
  task: Task;
  execute_at: number;
}

export interface QueueSnapshot {
  ready: Task[];
  delayed: DelayedEntry[];
}

export interface EnhancedMetrics extends Metrics {
  success_rate: number;
  delayed_queue_size: number;
  dead_letter_size: number;
  total_submitted: number;
}
