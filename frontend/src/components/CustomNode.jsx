import React, { useRef } from 'react';
import { Handle, Position } from '@xyflow/react';
import { Upload, FileText, Check, AlertTriangle, Download } from 'lucide-react';
import { TaskTideAPI } from '../api';

export const CustomNode = ({ data, selected, id }) => {
  const fileInputRef = useRef(null);
  const isReaderNode = data.type === 'csv.read' || data.type === 'doc.read';
  const isOutputNode = data.type === 'OUTPUT';

  // Execution state styles
  let stateRing = 'border-slate-200 dark:border-slate-800';
  let stateBg = 'bg-slate-50 dark:bg-zinc-900';
  let stateIcon = null;

  if (data.executionState === 'running') {
    stateRing = 'border-yellow-500 ring-1 ring-yellow-500/50 shadow-[0_0_8px_rgba(234,179,8,0.2)]';
    stateBg = 'bg-yellow-500/5 dark:bg-yellow-500/5';
  } else if (data.executionState === 'completed') {
    stateRing = 'border-green-600 ring-1 ring-green-600/50';
    stateBg = 'bg-green-600/5 dark:bg-green-600/5';
    stateIcon = <Check className="w-3.5 h-3.5 text-green-600" />;
  } else if (data.executionState === 'failed') {
    stateRing = 'border-red-600 ring-1 ring-red-600/50';
    stateBg = 'bg-red-600/5 dark:bg-red-600/5';
    stateIcon = <AlertTriangle className="w-3.5 h-3.5 text-red-600" />;
  } else if (selected) {
    stateRing = 'border-blue-500 ring-1 ring-blue-500/30';
  }

  const handleFileChange = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    if (data.onUploadStart) data.onUploadStart(id, file.name);

    try {
      const minioFile = await TaskTideAPI.uploadFile(file);
      if (data.onUploadSuccess) {
        data.onUploadSuccess(id, minioFile.id, file.name);
      }
    } catch (err) {
      console.error(err);
      if (data.onUploadError) data.onUploadError(id, err.message);
    }
  };

  const handleDownloadOutput = () => {
    if (data.outputFileId) {
      TaskTideAPI.downloadFile(data.outputFileId, `${data.label.replace(/\s+/g, '_')}_output.txt`);
    }
  };

  return (
    <div className={`w-[240px] text-xs bg-white dark:bg-zinc-950 border ${stateRing} shadow-sm select-none`}>
      {/* Target handle on the left (inputs) */}
      {!isReaderNode && (
        <Handle
          type="target"
          position={Position.Left}
          id="input"
          style={{ top: '50%' }}
        />
      )}

      {/* Node Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-100 dark:border-zinc-900 bg-slate-50 dark:bg-zinc-900/50">
        <div className="flex items-center gap-1.5 min-w-0">
          {stateIcon}
          <span className="font-medium truncate text-slate-800 dark:text-slate-100">
            {data.label}
          </span>
        </div>
        <span className="text-[10px] uppercase font-semibold text-slate-400 font-mono tracking-wider">
          {data.type.split('.').pop()}
        </span>
      </div>

      {/* Node Body */}
      <div className={`p-3 space-y-2.5 ${stateBg}`}>
        <div className="flex items-center justify-between text-[10px] text-slate-400 dark:text-zinc-500 font-mono">
          <span>TYPE</span>
          <span className="bg-slate-100 dark:bg-zinc-800 px-1 text-slate-600 dark:text-zinc-300 font-sans font-medium">
            {data.type}
          </span>
        </div>

        {/* Input file upload field (only for reader nodes) */}
        {isReaderNode && (
          <div className="mt-1 pt-1 border-t border-slate-100 dark:border-zinc-900">
            <input
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              className="hidden"
              accept={data.type === 'csv.read' ? '.csv' : '.txt,.doc,.docx,.pdf'}
            />
            {data.fileInputId ? (
              <div className="space-y-1.5">
                <div className="flex items-center gap-1.5 text-slate-600 dark:text-zinc-300 bg-slate-50 dark:bg-zinc-900 border border-slate-100 dark:border-zinc-800 px-2 py-1 truncate">
                  <FileText className="w-3.5 h-3.5 text-blue-500 shrink-0" />
                  <span className="truncate">{data.fileInputName || 'Attached File'}</span>
                </div>
                <button
                  type="button"
                  onClick={() => fileInputRef.current.click()}
                  className="w-full text-center py-1 text-[10px] border border-dashed border-slate-200 dark:border-zinc-800 text-slate-500 hover:text-slate-800 hover:border-slate-400 dark:hover:text-zinc-200 dark:hover:border-zinc-600 cursor-pointer"
                >
                  Replace File
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => fileInputRef.current.click()}
                className="w-full py-1.5 border border-dashed border-slate-200 dark:border-zinc-800 hover:border-slate-400 dark:hover:border-zinc-600 flex items-center justify-center gap-1 text-slate-500 hover:text-slate-800 dark:hover:text-zinc-200 cursor-pointer"
              >
                <Upload className="w-3.5 h-3.5" />
                {data.uploading ? 'Uploading...' : 'Upload File'}
              </button>
            )}
          </div>
        )}

        {/* Node output download button (when executed and has output file) */}
        {data.outputFileId && (
          <div className="pt-2 border-t border-slate-100 dark:border-zinc-900">
            <button
              type="button"
              onClick={handleDownloadOutput}
              className="w-full py-1 bg-blue-600 hover:bg-blue-700 text-white font-medium flex items-center justify-center gap-1 cursor-pointer transition-colors shadow-sm"
            >
              <Download className="w-3.5 h-3.5" />
              Download Output
            </button>
          </div>
        )}
      </div>

      {/* Source handle on the right (outputs) */}
      {!isOutputNode && (
        <Handle
          type="source"
          position={Position.Right}
          id="output"
          style={{ top: '50%' }}
        />
      )}
    </div>
  );
};
