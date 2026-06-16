import React from 'react';
import { Play, Save, Plus, Sun, Moon, Database } from 'lucide-react';

export const Header = ({
  workflows,
  selectedWorkflowId,
  onWorkflowSelect,
  onNew,
  onSave,
  onRun,
  isSaveDisabled,
  isRunDisabled,
  isDark,
  onToggleTheme,
  statusText,
  statusType
}) => {
  let statusDotClass = 'bg-slate-400 dark:bg-zinc-600';
  let statusTextColor = 'text-slate-500 dark:text-zinc-400';

  if (statusType === 'success') {
    statusDotClass = 'bg-green-600 shadow-[0_0_8px_rgba(22,163,74,0.5)]';
    statusTextColor = 'text-green-600 dark:text-green-500';
  } else if (statusType === 'running') {
    statusDotClass = 'bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.5)] animate-pulse';
    statusTextColor = 'text-yellow-600 dark:text-yellow-500';
  } else if (statusType === 'error') {
    statusDotClass = 'bg-red-600 shadow-[0_0_8px_rgba(220,38,38,0.5)]';
    statusTextColor = 'text-red-600 dark:text-red-500';
  }

  return (
    <header className="flex h-14 items-center justify-between px-4 border-b border-slate-200 dark:border-zinc-900 bg-white dark:bg-zinc-950 shrink-0">
      {/* Brand Logo */}
      <div className="flex items-center gap-2">
        <Database className="w-5 h-5 text-blue-600 dark:text-blue-500" />
        <div className="flex flex-col">
          <span className="text-sm font-bold tracking-tight text-slate-800 dark:text-slate-200 font-sans">
            Task<span className="text-blue-600 dark:text-blue-500">Tide</span>
          </span>
          <span className="text-[9px] text-slate-400 dark:text-zinc-500 font-mono tracking-widest -mt-0.5">
            ENGINE
          </span>
        </div>
      </div>

      {/* Control Actions */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <label htmlFor="workflow-dropdown" className="text-xs text-slate-500 dark:text-zinc-400 font-medium">
            Workflow:
          </label>
          <select
            id="workflow-dropdown"
            value={selectedWorkflowId || ''}
            onChange={(e) => onWorkflowSelect(e.target.value)}
            className="h-8 px-2 text-xs bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 focus:outline-none focus:border-slate-400 dark:focus:border-zinc-600 text-slate-800 dark:text-slate-100 cursor-pointer"
          >
            <option value="">-- Create or Select --</option>
            {workflows.map((w) => (
              <option key={w.id} value={w.id}>
                {w.workflow_name || `Unnamed (${w.id.substring(0, 8)})`}
              </option>
            ))}
          </select>
        </div>

        <button
          type="button"
          onClick={onNew}
          className="h-8 px-3 text-xs bg-slate-100 hover:bg-slate-200 dark:bg-zinc-900 dark:hover:bg-zinc-800 text-slate-700 dark:text-slate-200 font-medium flex items-center gap-1.5 cursor-pointer transition-colors border border-slate-200 dark:border-zinc-800"
        >
          <Plus className="w-3.5 h-3.5" />
          New
        </button>

        <button
          type="button"
          onClick={onSave}
          disabled={isSaveDisabled}
          className={`h-8 px-3 text-xs font-medium flex items-center gap-1.5 border transition-colors cursor-pointer ${
            isSaveDisabled
              ? 'opacity-40 cursor-not-allowed border-slate-200 dark:border-zinc-800 text-slate-400 dark:text-zinc-600'
              : 'bg-white hover:bg-slate-50 border-slate-200 dark:bg-zinc-900 dark:hover:bg-zinc-800 text-slate-800 dark:text-slate-200 dark:border-zinc-800'
          }`}
        >
          <Save className="w-3.5 h-3.5" />
          Save Definition
        </button>

        <button
          type="button"
          onClick={onRun}
          disabled={isRunDisabled}
          className={`h-8 px-4 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer shadow-sm ${
            isRunDisabled
              ? 'bg-slate-100 dark:bg-zinc-900 text-slate-400 dark:text-zinc-600 border border-slate-200 dark:border-zinc-800 opacity-40 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-700 text-white border border-blue-600'
          }`}
        >
          <Play className="w-3.5 h-3.5 fill-current" />
          Run Execution
        </button>
      </div>

      {/* Status Indicators & Theme */}
      <div className="flex items-center gap-4">
        {/* Status indicator */}
        <div className="flex items-center gap-2 border-r border-slate-200 dark:border-zinc-800 pr-4">
          <span className={`w-2 h-2 ${statusDotClass}`} />
          <span className={`text-xs font-medium font-mono ${statusTextColor}`}>
            {statusText}
          </span>
        </div>

        {/* Theme toggle */}
        <button
          type="button"
          onClick={onToggleTheme}
          className="p-1.5 hover:bg-slate-100 dark:hover:bg-zinc-800 border border-transparent hover:border-slate-200 dark:hover:border-zinc-800 text-slate-500 hover:text-slate-800 dark:text-zinc-400 dark:hover:text-zinc-100 cursor-pointer transition-colors"
          title="Toggle Light/Dark Theme"
        >
          {isDark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
        </button>
      </div>
    </header>
  );
};
