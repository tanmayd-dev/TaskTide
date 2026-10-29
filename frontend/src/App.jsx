import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  ReactFlow,
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  useReactFlow,
  ReactFlowProvider,
  MarkerType
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { TaskTideAPI } from './api';
import { Header } from './components/Header';
import { Sidebar } from './components/Sidebar';
import { CustomNode } from './components/CustomNode';
import { ExecutionMonitor } from './components/ExecutionMonitor';

const nodeTypes = {
  custom: CustomNode,
};

// Helper for generating unique client-side node IDs
const getUniqueId = () => `node_${Math.random().toString(36).substring(2, 9)}`;

function Workspace() {
  const reactFlowWrapper = useRef(null);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const { screenToFlowPosition } = useReactFlow();

  // State
  const [workflows, setWorkflows] = useState([]);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState(null);
  const [currentWorkflowName, setCurrentWorkflowName] = useState('');
  
  // Theme
  const [isDark, setIsDark] = useState(true);

  // Status Logger
  const [statusText, setStatusText] = useState('Backend Idle');
  const [statusType, setStatusType] = useState('info');

  // Execution Monitor State
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [executionId, setExecutionId] = useState(null);
  const [taskExecutions, setTaskExecutions] = useState([]);
  const executionIntervalRef = useRef(null);

  // Context Menu State
  const [menuVisible, setMenuVisible] = useState(false);
  const [menuPosition, setMenuPosition] = useState({ x: 0, y: 0 });
  const [menuScreenPos, setMenuScreenPos] = useState({ x: 0, y: 0 });

  // 1. Initial Load: Fetch workflow definitions
  const fetchWorkflows = async () => {
    try {
      const list = await TaskTideAPI.listWorkflows();
      setWorkflows(list);
    } catch (err) {
      console.error(err);
      setStatusText('Error connecting to backend API');
      setStatusType('error');
    }
  };

  useEffect(() => {
    fetchWorkflows();
  }, []);

  // Set theme class on document element
  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark);
  }, [isDark]);

  // Cleanup polling interval on unmount
  useEffect(() => {
    return () => {
      if (executionIntervalRef.current) clearInterval(executionIntervalRef.current);
    };
  }, []);

  const handleToggleTheme = () => {
    setIsDark(!isDark);
  };

  // Node File Upload Handlers (passed to nodes via data)
  const onUploadStart = useCallback((nodeId, fileName) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === nodeId) {
          return {
            ...node,
            data: { ...node.data, uploading: true, fileInputName: fileName }
          };
        }
        return node;
      })
    );
  }, [setNodes]);

  const onUploadSuccess = useCallback((nodeId, fileInputId, fileName) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === nodeId) {
          return {
            ...node,
            data: {
              ...node.data,
              uploading: false,
              fileInputId: fileInputId,
              fileInputName: fileName
            }
          };
        }
        return node;
      })
    );
    setStatusText('File uploaded successfully');
    setStatusType('success');
  }, [setNodes]);

  const onUploadError = useCallback((nodeId, errorMsg) => {
    setNodes((nds) =>
      nds.map((node) => {
        if (node.id === nodeId) {
          return {
            ...node,
            data: { ...node.data, uploading: false }
          };
        }
        return node;
      })
    );
    setStatusText(`File upload failed: ${errorMsg}`);
    setStatusType('error');
  }, [setNodes]);

  // 2. Add Node Function
  const addNode = useCallback((type, label, position, backendId = null, fileInputId = '') => {
    const id = backendId || getUniqueId();
    const newNode = {
      id,
      type: 'custom',
      position,
      data: {
        label: label || `${type} Node`,
        type,
        fileInputId,
        fileInputName: fileInputId ? `Attached File (${fileInputId.substring(0, 8)})` : '',
        executionState: 'idle',
        outputFileId: null,
        onUploadStart,
        onUploadSuccess,
        onUploadError
      }
    };
    setNodes((nds) => nds.concat(newNode));
    return id;
  }, [setNodes, onUploadStart, onUploadSuccess, onUploadError]);

  // Connect edges
  const onConnect = useCallback(
    (params) =>
      setEdges((eds) =>
        addEdge(
          {
            ...params,
            type: 'straight',
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: isDark ? '#3b82f6' : '#2563eb'
            }
          },
          eds
        )
      ),
    [setEdges, isDark]
  );

  // Drag and Drop support from Sidebar
  const onDragOver = useCallback((event) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event) => {
      event.preventDefault();

      const type = event.dataTransfer.getData('application/reactflow');
      if (!type) return;

      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY
      });

      const label = `${type} Node`;
      addNode(type, label, position);
    },
    [screenToFlowPosition, addNode]
  );

  // Context Menu handlers
  const onPaneContextMenu = useCallback(
    (event) => {
      event.preventDefault();
      event.stopPropagation();
      if (!reactFlowWrapper.current) return;
      const rect = reactFlowWrapper.current.getBoundingClientRect();
      setMenuVisible(true);
      setMenuScreenPos({
        x: event.clientX - rect.left,
        y: event.clientY - rect.top
      });
      setMenuPosition(
        screenToFlowPosition({
          x: event.clientX,
          y: event.clientY
        })
      );
    },
    [screenToFlowPosition]
  );

  const onNodeContextMenu = useCallback(
    (event) => {
      event.preventDefault();
      event.stopPropagation();
    },
    []
  );

  const onEdgeContextMenu = useCallback(
    (event, edge) => {
      event.preventDefault();
      event.stopPropagation();
      setEdges((eds) => eds.filter((e) => e.id !== edge.id));
      setStatusText('Connection deleted');
      setStatusType('info');
    },
    [setEdges]
  );

  const handleCloseMenu = () => {
    setMenuVisible(false);
  };

  useEffect(() => {
    if (menuVisible) {
      const handleClose = () => setMenuVisible(false);
      // Wait for current execution frame to clear so that the initial contextmenu/click doesn't trigger this listener immediately
      const timer = setTimeout(() => {
        document.addEventListener('click', handleClose);
      }, 0);
      return () => {
        clearTimeout(timer);
        document.removeEventListener('click', handleClose);
      };
    }
  }, [menuVisible]);

  // Select Workflow dropdown handler
  const handleWorkflowSelect = async (workflowId) => {
    if (!workflowId) {
      setSelectedWorkflowId(null);
      setCurrentWorkflowName('');
      setNodes([]);
      setEdges([]);
      return;
    }

    try {
      setStatusText('Loading workflow...');
      setStatusType('running');

      const fatWorkflow = await TaskTideAPI.getWorkflow(workflowId);
      const tasks = fatWorkflow.workflow_tasks || [];

      setSelectedWorkflowId(workflowId);
      setCurrentWorkflowName(fatWorkflow.workflow_name);

      // Map backend tasks to nodes
      const newNodes = [];
      const newEdges = [];

      tasks.forEach((task, idx) => {
        const col = idx % 4;
        const row = Math.floor(idx / 4);
        const position = { x: 100 + col * 280, y: 120 + row * 180 };

        newNodes.push({
          id: task.id,
          type: 'custom',
          position,
          data: {
            label: task.task_name,
            type: task.task_type,
            fileInputId: task.task_input || '',
            fileInputName: task.task_input ? `Attached File (${task.task_input.substring(0, 8)})` : '',
            executionState: 'idle',
            outputFileId: null,
            onUploadStart,
            onUploadSuccess,
            onUploadError
          }
        });
      });

      // Map dependencies to edges
      tasks.forEach((task) => {
        if (task.task_input_task) {
          newEdges.push({
            id: `edge_${task.task_input_task}_to_${task.id}`,
            source: task.task_input_task,
            target: task.id,
            type: 'straight',
            markerEnd: {
              type: MarkerType.ArrowClosed,
              color: isDark ? '#3b82f6' : '#2563eb'
            }
          });
        }
      });

      setNodes(newNodes);
      setEdges(newEdges);
      setStatusText('Workflow loaded');
      setStatusType('success');
    } catch (err) {
      console.error(err);
      setStatusText('Failed to load workflow');
      setStatusType('error');
    }
  };

  // 3. New Workflow Action
  const handleNewWorkflow = () => {
    const name = prompt('Enter a name for the new workflow:', 'My Custom Workflow');
    if (!name || !name.trim()) return;

    setNodes([]);
    setEdges([]);
    setSelectedWorkflowId(null);
    setCurrentWorkflowName(name.trim());

    // Add starting reader and output nodes
    const readerId = addNode('csv.read', 'Read Tasks', { x: 100, y: 150 });
    const outputId = addNode('OUTPUT', 'Engine Output', { x: 450, y: 150 });

    // Link them
    setEdges([
      {
        id: `edge_${readerId}_to_${outputId}`,
        source: readerId,
        target: outputId,
        type: 'straight',
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: isDark ? '#3b82f6' : '#2563eb'
        }
      }
    ]);

    setStatusText(`Created workflow draft: "${name.trim()}". Save to persist.`);
    setStatusType('success');
  };

  // Topological sorting helper
  const topologicalSort = (nodesList, edgesList) => {
    const sorted = [];
    const visited = new Set();
    const temp = new Set();

    const visit = (nodeId) => {
      if (temp.has(nodeId)) {
        throw new Error('Cyclic dependency detected in workflow connections!');
      }
      if (!visited.has(nodeId)) {
        temp.add(nodeId);
        
        // Find nodes feeding into nodeId (incoming edges)
        const incomingEdges = edgesList.filter((e) => e.target === nodeId);
        for (const edge of incomingEdges) {
          visit(edge.source);
        }
        
        temp.delete(nodeId);
        visited.add(nodeId);
        
        const node = nodesList.find((n) => n.id === nodeId);
        if (node) sorted.push(node);
      }
    };

    for (const node of nodesList) {
      visit(node.id);
    }
    return sorted;
  };

  // 4. Save Workflow Action
  const handleSaveWorkflow = async () => {
    if (!currentWorkflowName) return;

    try {
      setStatusText('Saving workflow...');
      setStatusType('running');

      // Sort nodes topologically
      let sortedNodes;
      try {
        sortedNodes = topologicalSort(nodes, edges);
      } catch (err) {
        setStatusText(err.message);
        setStatusType('error');
        return;
      }

      // Create workflow definition
      const workflow = await TaskTideAPI.createWorkflow(currentWorkflowName);
      const workflowId = workflow.id;

      const clientToBackendMap = {};

      // Add tasks sequentially
      for (const node of sortedNodes) {
        // Find incoming edge to determine task input task
        let inputTaskId = '';
        const incomingEdge = edges.find((e) => e.target === node.id);
        if (incomingEdge) {
          inputTaskId = clientToBackendMap[incomingEdge.source] || '';
        }

        const taskPayload = {
          task_type: node.data.type,
          task_name: node.data.label,
          workflow_id: workflowId,
          task_input: node.data.fileInputId || '',
          task_input_task: inputTaskId
        };

        const createdTask = await TaskTideAPI.addTask(taskPayload);
        clientToBackendMap[node.id] = createdTask.id;
      }

      setSelectedWorkflowId(workflowId);
      await fetchWorkflows();

      setStatusText('Workflow saved successfully!');
      setStatusType('success');
    } catch (err) {
      console.error(err);
      setStatusText(`Save failed: ${err.message}`);
      setStatusType('error');
    }
  };

  // Poll workflow execution run progress
  const startExecutionPolling = (execId) => {
    if (executionIntervalRef.current) clearInterval(executionIntervalRef.current);

    executionIntervalRef.current = setInterval(async () => {
      try {
        const tasks = await TaskTideAPI.getWorkflowStatus(execId);
        setTaskExecutions(tasks);

        // Update node execution state visuals
        setNodes((nds) =>
          nds.map((node) => {
            const exec = tasks.find((t) => t.task_base === node.id);
            if (exec) {
              let executionState = 'idle';
              if (exec.task_completed) {
                executionState = 'completed';
              } else if (exec.task_assigned) {
                executionState = 'running';
              }

              return {
                ...node,
                data: {
                  ...node.data,
                  executionState,
                  outputFileId: exec.task_completed ? exec.task_output : null
                }
              };
            }
            return node;
          })
        );

        // Check if all tasks finished
        const allCompleted = tasks.length > 0 && tasks.every((t) => t.task_completed);
        if (allCompleted) {
          setStatusText('Execution Run Completed!');
          setStatusType('success');
          if (executionIntervalRef.current) {
            clearInterval(executionIntervalRef.current);
            executionIntervalRef.current = null;
          }
        }
      } catch (err) {
        console.error('Status polling error:', err);
      }
    }, 1000);
  };

  // 5. Run Execution Action
  const handleRunWorkflow = async () => {
    if (!selectedWorkflowId) return;

    try {
      setStatusText('Triggering workflow run...');
      setStatusType('running');

      const run = await TaskTideAPI.startWorkflow(selectedWorkflowId);
      const execId = run.id;

      setExecutionId(execId);
      setTaskExecutions([]);
      setIsDrawerOpen(true);

      // Start polling status
      startExecutionPolling(execId);
    } catch (err) {
      console.error(err);
      setStatusText(`Run failed: ${err.message}`);
      setStatusType('error');
    }
  };

  const handleDeleteSelected = () => {
    const selectedNodeIds = nodes.filter(n => n.selected).map(n => n.id);
    if (selectedNodeIds.length > 0) {
      setNodes((nds) => nds.filter((node) => !node.selected));
      setEdges((eds) => eds.filter((edge) => !edge.selected && !selectedNodeIds.includes(edge.source) && !selectedNodeIds.includes(edge.target)));
      setStatusText('Selected node(s) deleted');
      setStatusType('info');
      return;
    }

    const selectedEdges = edges.filter(e => e.selected);
    if (selectedEdges.length > 0) {
      setEdges((eds) => eds.filter((edge) => !edge.selected));
      setStatusText('Selected connection(s) deleted');
      setStatusType('info');
      return;
    }

    setStatusText('No node or edge selected to delete');
    setStatusType('error');
  };

  const handleClearAll = () => {
    if (window.confirm("Are you sure you want to clear the entire workspace?")) {
      setNodes([]);
      setEdges([]);
      setSelectedWorkflowId(null);
      setCurrentWorkflowName('');
      setStatusText('Workspace cleared');
      setStatusType('info');
    }
  };

  const handleCloseDrawer = () => {
    setIsDrawerOpen(false);
    if (executionIntervalRef.current) {
      clearInterval(executionIntervalRef.current);
      executionIntervalRef.current = null;
    }
  };

  return (
    <div className="flex flex-col h-screen w-screen bg-slate-50 dark:bg-zinc-950 text-slate-800 dark:text-zinc-100">
      <Header
        workflows={workflows}
        selectedWorkflowId={selectedWorkflowId}
        onWorkflowSelect={handleWorkflowSelect}
        onNew={handleNewWorkflow}
        onSave={handleSaveWorkflow}
        onRun={handleRunWorkflow}
        onDeleteSelected={handleDeleteSelected}
        onClearAll={handleClearAll}
        isSaveDisabled={!currentWorkflowName || nodes.length === 0}
        isRunDisabled={!selectedWorkflowId}
        isDark={isDark}
        onToggleTheme={handleToggleTheme}
        statusText={statusText}
        statusType={statusType}
      />

      <div className="flex flex-1 overflow-hidden relative">
        <Sidebar />

        <div ref={reactFlowWrapper} className="flex-1 h-full relative bg-slate-50 dark:bg-zinc-950">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            onDrop={onDrop}
            onDragOver={onDragOver}
            onPaneContextMenu={onPaneContextMenu}
            onNodeContextMenu={onNodeContextMenu}
            onEdgeContextMenu={onEdgeContextMenu}
            nodeTypes={nodeTypes}
            snapToGrid
            snapGrid={[15, 15]}
            fitView
          >
            <Controls className="dark:bg-zinc-900 border-zinc-200 dark:border-zinc-800 text-foreground" />
            <MiniMap
              nodeColor={() => (isDark ? '#27272a' : '#e4e4e7')}
              maskColor={isDark ? 'rgba(9,9,11, 0.7)' : 'rgba(255,255,255, 0.7)'}
              className="border border-zinc-200 dark:border-zinc-800"
            />
            <Background
              variant="dots"
              gap={15}
              size={1.5}
              color={isDark ? '#27272a' : '#e4e4e7'}
            />
          </ReactFlow>

          {/* Quick Context Menu */}
          {menuVisible && (
            <div
              className="absolute z-50 bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 shadow-lg text-xs py-1 min-w-[150px]"
              style={{
                top: menuScreenPos.y,
                left: menuScreenPos.x
              }}
            >
              <div className="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-slate-400 dark:text-zinc-500 font-mono border-b border-slate-100 dark:border-zinc-800">
                Add Task Node
              </div>
              {[
                { type: 'csv.read', label: 'CSV Read' },
                { type: 'doc.read', label: 'Doc Read' },
                { type: 'NOOP', label: 'NOOP' },
                { type: 'doc.check_sensitive', label: 'Check Sensitive' },
                { type: 'csv.validate', label: 'CSV Validate' },
                { type: 'csv.aggregate', label: 'CSV Aggregate' },
                { type: 'csv.filter', label: 'CSV Filter' },
                { type: 'OUTPUT', label: 'OUTPUT Node' }
              ].map((item) => (
                <button
                  key={item.type}
                  type="button"
                  onClick={() => addNode(item.type, item.label, menuPosition)}
                  className="w-full text-left px-3 py-1.5 hover:bg-slate-100 dark:hover:bg-zinc-800 text-slate-700 dark:text-slate-200 cursor-pointer"
                >
                  {item.label}
                </button>
              ))}
            </div>
          )}
        </div>

        <ExecutionMonitor
          isOpen={isDrawerOpen}
          onClose={handleCloseDrawer}
          executionId={executionId}
          status={
            taskExecutions.length > 0 && taskExecutions.every((t) => t.task_completed)
              ? 'completed'
              : 'running'
          }
          taskExecutions={taskExecutions}
          nodes={nodes}
        />
      </div>
    </div>
  );
}

export default function App() {
  return (
    <ReactFlowProvider>
      <Workspace />
    </ReactFlowProvider>
  );
}
