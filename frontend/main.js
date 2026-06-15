// main.js - Application Entrypoint and UI Controller

import { TaskTideAPI } from './api.js';
import { NodeEditor } from './nodes.js';

document.addEventListener('DOMContentLoaded', () => {
    // Initialize Node Editor
    const editor = new NodeEditor('canvas-container', 'svg-connections', 'nodes-container');
    
    // UI Elements
    const workflowSelect = document.getElementById('workflow-select');
    const btnNew = document.getElementById('btn-new');
    const btnSave = document.getElementById('btn-save');
    const btnRun = document.getElementById('btn-run');
    const contextMenu = document.getElementById('context-menu');
    
    const executionDrawer = document.getElementById('execution-drawer');
    const btnCloseDrawer = document.getElementById('btn-close-drawer');
    const executionIdText = document.getElementById('execution-id-text');
    const executionStatusBadge = document.getElementById('execution-status-badge');
    const executionTasksList = document.getElementById('execution-tasks-list');
    
    // Active State variables
    let currentWorkflowId = null;
    let currentWorkflowName = '';
    let menuPosition = { x: 0, y: 0 };
    let executionInterval = null;
    
    // 1. Initial Load: Populate workflow definitions list
    async function loadWorkflowDropdown() {
        try {
            const list = await TaskTideAPI.listWorkflows();
            
            // Clear current options (except default)
            workflowSelect.innerHTML = '<option value="">-- Create or Select Workflow --</option>';
            
            list.forEach(w => {
                const opt = document.createElement('option');
                opt.value = w.id;
                opt.textContent = w.workflow_name || `Unnamed (${w.id.substr(0, 8)})`;
                workflowSelect.appendChild(opt);
            });
            
            // Reselect active if present
            if (currentWorkflowId) {
                workflowSelect.value = currentWorkflowId;
            }
        } catch (err) {
            console.error('Dropdown load failed:', err);
            editor.showStatus('Error connecting to backend API', 'error');
        }
    }
    
    loadWorkflowDropdown();
    
    // 2. Dropdown Change Handler: Load existing workflow
    workflowSelect.addEventListener('change', async (e) => {
        const id = e.target.value;
        if (!id) {
            currentWorkflowId = null;
            currentWorkflowName = '';
            editor.clear();
            btnSave.disabled = true;
            btnRun.disabled = true;
            return;
        }
        
        currentWorkflowId = id;
        currentWorkflowName = workflowSelect.options[workflowSelect.selectedIndex].text;
        
        await editor.loadWorkflow(id);
        
        btnSave.disabled = false;
        btnRun.disabled = false;
    });
    
    // 3. New Workflow Handler
    btnNew.addEventListener('click', () => {
        const name = prompt('Enter name for the new workflow:', 'My Custom Workflow');
        if (!name || !name.trim()) return;
        
        editor.clear();
        currentWorkflowId = null; // resets ID to force create on save
        currentWorkflowName = name.trim();
        
        // Add a default CSV Read node and Output node to guide the user
        editor.addNode('csv.read', 'Read Tasks', 150, 200);
        editor.addNode('OUTPUT', 'Engine Output', 550, 200);
        
        btnSave.disabled = false;
        btnRun.disabled = true; // Need to save first to get ID
        
        editor.showStatus(`Created workflow draft: "${currentWorkflowName}". Click Save.`, 'info');
    });
    
    // 4. Save Workflow Handler
    btnSave.addEventListener('click', async () => {
        if (!currentWorkflowName) return;
        
        try {
            const id = await editor.saveWorkflow(currentWorkflowName);
            currentWorkflowId = id;
            btnRun.disabled = false;
            
            await loadWorkflowDropdown();
            editor.showStatus('Workflow saved and ready to run!', 'success');
        } catch (err) {
            console.error(err);
        }
    });
    
    // 5. Run Execution Handler
    btnRun.addEventListener('click', async () => {
        if (!currentWorkflowId) return;
        
        try {
            editor.showStatus('Triggering workflow run...', 'running');
            const run = await TaskTideAPI.startWorkflow(currentWorkflowId);
            const executionId = run.id;
            
            // Show execution monitor drawer
            openExecutionDrawer(executionId);
            
            // Start Polling Status
            startExecutionPolling(executionId);
        } catch (err) {
            console.error(err);
            editor.showStatus(`Run failed: ${err.message}`, 'error');
        }
    });
    
    // Context Menu handling for Node Creation
    const canvasContainer = document.getElementById('canvas-container');
    
    canvasContainer.addEventListener('contextmenu', (e) => {
        e.preventDefault();
        
        // Keep screen coordinate position for drawing context menu
        contextMenu.style.left = `${e.clientX}px`;
        contextMenu.style.top = `${e.clientY}px`;
        contextMenu.style.display = 'block';
        
        // Save canvas coordinates position where click occurred to draw node
        menuPosition = editor.getCanvasCoordinates(e.clientX, e.clientY);
    });
    
    // Hide context menu on left click outside
    document.addEventListener('click', (e) => {
        if (!contextMenu.contains(e.target)) {
            contextMenu.style.display = 'none';
        }
    });
    
    // Menu item clicks (Node types choices)
    contextMenu.addEventListener('click', (e) => {
        const item = e.target.closest('.menu-item');
        if (!item) return;
        
        const type = item.dataset.type;
        const name = `${type} Node`;
        
        editor.addNode(type, name, menuPosition.x, menuPosition.y);
        editor.drawAllConnections();
        
        contextMenu.style.display = 'none';
    });
    
    // Drawer handlers
    btnCloseDrawer.addEventListener('click', () => {
        executionDrawer.style.display = 'none';
        stopExecutionPolling();
    });
    
    function openExecutionDrawer(executionId) {
        executionDrawer.style.display = 'flex';
        executionIdText.textContent = executionId;
        executionStatusBadge.className = 'badge running';
        executionStatusBadge.textContent = 'Running';
        executionTasksList.innerHTML = '<div class="empty-state">Initializing task queue...</div>';
    }
    
    function startExecutionPolling(executionId) {
        stopExecutionPolling(); // clear previous
        
        // Poll status every 1 second
        executionInterval = setInterval(async () => {
            try {
                const tasks = await TaskTideAPI.getWorkflowStatus(executionId);
                renderExecutionTasks(tasks);
                
                // Update graph visual status highlights
                editor.updateTaskStates(tasks);
                
                // Check if all completed
                const allCompleted = tasks.length > 0 && tasks.every(t => t.task_completed);
                if (allCompleted) {
                    executionStatusBadge.className = 'badge completed';
                    executionStatusBadge.textContent = 'Completed';
                    editor.showStatus('Execution Run Completed!', 'success');
                    stopExecutionPolling();
                }
            } catch (err) {
                console.error(err);
            }
        }, 1000);
    }
    
    function stopExecutionPolling() {
        if (executionInterval) {
            clearInterval(executionInterval);
            executionInterval = null;
        }
    }
    
    function renderExecutionTasks(tasks) {
        if (tasks.length === 0) {
            executionTasksList.innerHTML = '<div class="empty-state">Queue scheduled. Waiting for worker...</div>';
            return;
        }
        
        executionTasksList.innerHTML = '';
        tasks.forEach(task => {
            const node = editor.nodes.find(n => n.id === task.task_base);
            const taskName = node ? node.name : `Task (${task.task_base.substr(0,8)})`;
            
            const item = document.createElement('div');
            item.className = 'task-exec-item';
            
            let statusBadge = '';
            if (task.task_completed) {
                statusBadge = '<span class="badge completed">Completed</span>';
            } else if (task.task_assigned) {
                statusBadge = '<span class="badge running">Running</span>';
            } else {
                statusBadge = '<span class="badge idle">Pending</span>';
            }
            
            item.innerHTML = `
                <div class="task-exec-title">${taskName}</div>
                <div class="task-exec-status">
                    <span class="code-text" style="font-size: 9px; padding: 2px 4px;">ID: ${task.id.substr(0, 8)}</span>
                    ${statusBadge}
                </div>
            `;
            executionTasksList.appendChild(item);
        });
    }
});
