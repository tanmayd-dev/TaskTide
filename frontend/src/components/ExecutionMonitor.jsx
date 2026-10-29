import React from 'react';
import { X, CheckCircle2, Play, AlertCircle, Clock, Download } from 'lucide-react';
import { TaskTideAPI } from '../api';

export const ExecutionMonitor = ({
  isOpen,
  onClose,
  executionId,
  status,
  taskExecutions,
  nodes,
  edges
}) => {
  if (!isOpen) return null;

  const getTaskName = (taskBaseId) => {
    const node = nodes.find(n => n.id === taskBaseId);
    return node ? node.data.label : `Task (${taskBaseId.substring(0, 8)})`;
  };

  const getNodeType = (taskBaseId) => {
    const node = nodes.find(n => n.id === taskBaseId);
    return node ? node.data.type : null;
  };

  // Order executions along the canvas DAG (topological order) instead of
  // raw backend row order. Executions not present on the canvas keep their
  // backend order at the end of the list.
  const sortByWorkflowOrder = (execs) => {
    const edgesList = Array.isArray(edges) ? edges : [];
    const known = new Map();

    nodes.forEach((n) => known.set(n.id, true));

    const dependents = new Map();
    const indegree = new Map();
    nodes.forEach((n) => indegree.set(n.id, 0));
    edgesList.forEach((e) => {
      if (!known.has(e.source) || !known.has(e.target)) return;
      dependents.set(e.source, [...(dependents.get(e.source) || []), e.target]);
      indegree.set(e.target, (indegree.get(e.target) || 0) + 1);
    });

    // Kahn's algorithm; cycle remnants are appended in backend order.
    const ordered = [];
    const queue = nodes
      .filter((n) => (indegree.get(n.id) || 0) === 0)
      .map((n) => n.id);
    const visited = new Set();
    while (queue.length > 0) {
      const id = queue.shift();
      if (visited.has(id)) continue;
      visited.add(id);
      ordered.push(id);
      for (const nextId of dependents.get(id) || []) {
        const remaining = indegree.get(nextId) - 1;
        indegree.set(nextId, remaining);
        if (remaining === 0) queue.push(nextId);
      }
    }
    nodes.forEach((n) => {
      if (!visited.has(n.id)) ordered.push(n.id);
    });

    const rank = new Map(ordered.map((id, index) => [id, index]));
    return [...execs].sort(
      (a, b) =>
        (rank.get(a.task_base) ?? Number.MAX_SAFE_INTEGER) -
          (rank.get(b.task_base) ?? Number.MAX_SAFE_INTEGER) ||
        String(a.id).localeCompare(String(b.id))
    );
  };

  const sortedExecutions = sortByWorkflowOrder(taskExecutions || []);

  const handleDownloadOutput = (outputFileId, taskName) => {
    if (outputFileId) {
      TaskTideAPI.downloadFile(outputFileId, `${taskName.replace(/\s+/g, '_')}_output.txt`);
    }
  };

  let statusBadge = (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium bg-slate-100 text-slate-800 dark:bg-zinc-900 dark:text-zinc-300 border border-slate-200 dark:border-zinc-800">
      <Clock className="w-3.5 h-3.5" />
      Idle
    </span>
  );

  if (status === 'running') {
    statusBadge = (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-semibold bg-yellow-500/10 text-yellow-600 dark:text-yellow-500 border border-yellow-500/20 animate-pulse">
        <Play className="w-3 h-3 fill-current" />
        Running
      </span>
    );
  } else if (status === 'completed') {
    statusBadge = (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-semibold bg-green-500/10 text-green-600 dark:text-green-500 border border-green-500/20">
        <CheckCircle2 className="w-3.5 h-3.5" />
        Completed
      </span>
    );
  } else if (status === 'failed') {
    statusBadge = (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-semibold bg-red-500/10 text-red-600 dark:text-red-500 border border-red-500/20">
        <AlertCircle className="w-3.5 h-3.5" />
        Failed
      </span>
    );
  }

  return (
    <div className="w-80 border-l border-slate-200 dark:border-zinc-900 bg-white dark:bg-zinc-950 flex flex-col shrink-0 h-full z-50">
      {/* Drawer Header */}
      <div className="flex h-14 items-center justify-between px-4 border-b border-slate-200 dark:border-zinc-900">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-800 dark:text-slate-200 font-mono">
          Run Monitor
        </h3>
        <button
          type="button"
          onClick={onClose}
          className="p-1 text-slate-500 hover:text-slate-800 dark:text-zinc-400 dark:hover:text-zinc-100 cursor-pointer"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Drawer Body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5">
        {/* Execution Metadata */}
        <div className="space-y-3 bg-slate-50/50 dark:bg-zinc-900/40 p-3 border border-slate-200 dark:border-zinc-800">
          <div className="space-y-1">
            <span className="text-[10px] font-bold text-slate-400 dark:text-zinc-500 uppercase tracking-widest font-mono">
              Execution ID
            </span>
            <div className="code-text font-mono break-all text-slate-700 dark:text-zinc-300">
              {executionId || 'N/A'}
            </div>
          </div>
          <div className="flex items-center justify-between pt-1">
            <span className="text-[10px] font-bold text-slate-400 dark:text-zinc-500 uppercase tracking-widest font-mono">
              Status
            </span>
            {statusBadge}
          </div>
        </div>

        {/* Task Executions List */}
        <div className="space-y-2">
          <span className="text-[10px] font-bold text-slate-400 dark:text-zinc-500 uppercase tracking-widest font-mono block">
            Task Execution List
          </span>
          
          <div className="flex flex-col gap-2">
            {sortedExecutions.length === 0 ? (
              <div className="text-xs text-slate-400 dark:text-zinc-600 text-center py-8 border border-dashed border-slate-200 dark:border-zinc-900">
                Waiting for queue assignment...
              </div>
            ) : (
              sortedExecutions.map((task, index) => {
                const name = getTaskName(task.task_base);
                const nodeType = getNodeType(task.task_base);
                let taskStateBadge = (
                  <span className="px-1.5 py-0.5 text-[9px] font-mono font-medium bg-slate-100 text-slate-600 dark:bg-zinc-900 dark:text-zinc-400 border border-slate-200 dark:border-zinc-800">
                    PENDING
                  </span>
                );

                if (task.task_completed) {
                  taskStateBadge = (
                    <span className="px-1.5 py-0.5 text-[9px] font-mono font-semibold bg-green-500/10 text-green-600 dark:text-green-500 border border-green-500/20">
                      SUCCESS
                    </span>
                  );
                } else if (task.task_failed) {
                  taskStateBadge = (
                    <span className="px-1.5 py-0.5 text-[9px] font-mono font-semibold bg-red-500/10 text-red-600 dark:text-red-500 border border-red-500/20">
                      FAILED
                    </span>
                  );
                } else if (task.task_assigned) {
                  taskStateBadge = (
                    <span className="px-1.5 py-0.5 text-[9px] font-mono font-semibold bg-yellow-500/10 text-yellow-600 dark:text-yellow-500 border border-yellow-500/20 animate-pulse">
                      RUNNING
                    </span>
                  );
                }

                return (
                  <div
                    key={task.id}
                    className="p-3 border border-slate-200 dark:border-zinc-900 bg-white dark:bg-zinc-950 flex flex-col gap-2 shadow-sm"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-[9px] font-mono font-bold uppercase tracking-widest text-slate-400 dark:text-zinc-500">
                        Step {index + 1}
                      </span>
                      {taskStateBadge}
                    </div>

                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="font-semibold text-xs text-slate-800 dark:text-slate-200 truncate">
                          {name}
                        </div>
                        <div className="flex items-center gap-1.5 mt-0.5">
                          {nodeType && (
                            <span className="text-[9px] font-mono uppercase tracking-wider px-1 py-0.5 bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-500/20">
                              {nodeType}
                            </span>
                          )}
                          <span className="text-[10px] text-slate-400 dark:text-zinc-500 font-mono">
                            ID: {task.id.substring(0, 8)}
                          </span>
                        </div>
                      </div>
                    </div>

                    {/* Output Download link if completed */}
                    {task.task_completed && task.task_output && (
                      <button
                        type="button"
                        onClick={() => handleDownloadOutput(task.task_output, name)}
                        className="w-full py-1 border border-slate-200 hover:border-slate-400 dark:border-zinc-800 dark:hover:border-zinc-600 hover:bg-slate-50 dark:hover:bg-zinc-900 flex items-center justify-center gap-1.5 text-[10px] font-medium text-slate-700 dark:text-zinc-300 cursor-pointer transition-all"
                      >
                        <Download className="w-3 h-3 text-blue-500" />
                        Download Node Output
                      </button>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
