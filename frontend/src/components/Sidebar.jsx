import React from 'react';
import { PlaySquare, FileInput, ShieldAlert, FileOutput, HelpCircle } from 'lucide-react';

const NODE_TYPES = [
  { type: 'csv.read', label: 'CSV Read', category: 'Source', icon: <FileInput className="w-4 h-4 text-blue-500" /> },
  { type: 'doc.read', label: 'Doc Read', category: 'Source', icon: <FileInput className="w-4 h-4 text-blue-500" /> },
  { type: 'NOOP', label: 'NOOP Task', category: 'Process', icon: <PlaySquare className="w-4 h-4 text-slate-500" /> },
  { type: 'doc.check_sensitive', label: 'Check Sensitive', category: 'Process', icon: <ShieldAlert className="w-4 h-4 text-red-500" /> },
  { type: 'csv.validate', label: 'CSV Validate', category: 'Process', icon: <PlaySquare className="w-4 h-4 text-slate-500" /> },
  { type: 'csv.aggregate', label: 'CSV Aggregate', category: 'Process', icon: <PlaySquare className="w-4 h-4 text-slate-500" /> },
  { type: 'csv.filter', label: 'CSV Filter', category: 'Process', icon: <PlaySquare className="w-4 h-4 text-slate-500" /> },
  { type: 'OUTPUT', label: 'Engine Output', category: 'Output', icon: <FileOutput className="w-4 h-4 text-green-600" /> },
];

export const Sidebar = () => {
  const onDragStart = (event, nodeType) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <aside className="w-60 border-r border-slate-200 dark:border-zinc-900 bg-white dark:bg-zinc-950 p-4 flex flex-col gap-4 shrink-0 h-full overflow-y-auto select-none">
      <div className="space-y-1">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400 dark:text-zinc-500 font-mono">
          Node Library
        </h3>
        <p className="text-[11px] text-slate-500 dark:text-zinc-400 leading-normal">
          Drag and drop nodes onto the canvas to assemble your workflow pipelines.
        </p>
      </div>

      <div className="border-t border-slate-100 dark:border-zinc-900 pt-3 flex flex-col gap-3">
        {/* Group by category */}
        {['Source', 'Process', 'Output'].map((category) => {
          const nodes = NODE_TYPES.filter((n) => n.category === category);
          if (nodes.length === 0) return null;

          return (
            <div key={category} className="space-y-1.5">
              <span className="text-[10px] font-bold text-slate-400 dark:text-zinc-500 uppercase tracking-widest font-mono">
                {category}s
              </span>
              <div className="flex flex-col gap-1.5">
                {nodes.map((node) => (
                  <div
                    key={node.type}
                    onDragStart={(event) => onDragStart(event, node.type)}
                    draggable
                    className="flex items-center gap-2 px-3 py-2 border border-slate-200 dark:border-zinc-800 bg-slate-50/50 hover:bg-slate-100/50 dark:bg-zinc-900/40 dark:hover:bg-zinc-900/80 cursor-grab active:cursor-grabbing text-xs text-slate-700 dark:text-slate-200 transition-colors shadow-sm"
                  >
                    {node.icon}
                    <div className="flex flex-col">
                      <span className="font-medium">{node.label}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      <div className="mt-auto border-t border-slate-100 dark:border-zinc-900 pt-3">
        <div className="flex gap-2 text-[10px] text-slate-500 dark:text-zinc-500 bg-slate-50 dark:bg-zinc-900/50 p-2.5 border border-slate-200 dark:border-zinc-800 font-mono">
          <HelpCircle className="w-4 h-4 text-blue-500 shrink-0" />
          <div className="space-y-1 font-sans">
            <span className="font-semibold text-slate-600 dark:text-zinc-400">Quick Tips:</span>
            <ul className="list-disc pl-3.5 space-y-0.5 text-[9px] text-slate-500 dark:text-zinc-500">
              <li>Right-click canvas to search-add nodes</li>
              <li>Right-click node or edge to delete it</li>
              <li>Press Save to persist definition</li>
            </ul>
          </div>
        </div>
      </div>
    </aside>
  );
};
