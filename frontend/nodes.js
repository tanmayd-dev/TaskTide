// nodes.js - ComfyUI-style Workflow Graph Engine

import { TaskTideAPI } from './api.js';

export class NodeEditor {
    constructor(containerId, svgId, nodesContainerId) {
        this.container = document.getElementById(containerId);
        this.svg = document.getElementById(svgId);
        this.connectionsGroup = document.getElementById('connections-group');
        this.nodesContainer = document.getElementById(nodesContainerId);
        this.tempWire = document.getElementById('temp-wire');
        
        this.nodes = [];
        this.connections = []; // List of { id, fromNode, fromPort, toNode, toPort }
        
        // Pan and Zoom State
        this.panX = 100;
        this.panY = 150;
        this.zoom = 1.0;
        
        // Dragging Canvas State
        this.isPanning = false;
        this.startX = 0;
        this.startY = 0;
        this.isSpacePressed = false;
        
        // Dragging Node State
        this.draggedNode = null;
        this.nodeOffsetX = 0;
        this.nodeOffsetY = 0;
        
        // Dragging Wire State
        this.draggingWire = null; // { fromNode, fromPort, startX, startY }
        
        // Selection State
        this.selectedNode = null;
        
        this.initEvents();
        this.updateCanvasTransform();
    }
    
    initEvents() {
        // Space key check for panning
        window.addEventListener('keydown', (e) => {
            if (e.code === 'Space') {
                this.isSpacePressed = true;
                this.container.style.cursor = 'grab';
            }
        });
        
        window.addEventListener('keyup', (e) => {
            if (e.code === 'Space') {
                this.isSpacePressed = false;
                this.container.style.cursor = 'default';
            }
        });
        
        // Canvas mouse down for panning
        this.container.addEventListener('mousedown', (e) => {
            if (e.button === 1 || (e.button === 0 && this.isSpacePressed)) {
                e.preventDefault();
                this.isPanning = true;
                this.startX = e.clientX - this.panX;
                this.startY = e.clientY - this.panY;
                this.container.style.cursor = 'grabbing';
            }
        });
        
        // Mouse move handles canvas pan, node drag, and wire drag
        window.addEventListener('mousemove', (e) => {
            if (this.isPanning) {
                this.panX = e.clientX - this.startX;
                this.panY = e.clientY - this.startY;
                this.updateCanvasTransform();
            } else if (this.draggedNode) {
                // Drag node inside canvas coordinates
                const rect = this.nodesContainer.getBoundingClientRect();
                const x = (e.clientX - rect.left - this.nodeOffsetX) / this.zoom;
                const y = (e.clientY - rect.top - this.nodeOffsetY) / this.zoom;
                
                // Snap to grid of 10px
                this.draggedNode.x = Math.round(x / 10) * 10;
                this.draggedNode.y = Math.round(y / 10) * 10;
                
                this.updateNodePosition(this.draggedNode);
                this.drawAllConnections();
            } else if (this.draggingWire) {
                // Drag wire
                const mousePos = this.getCanvasCoordinates(e.clientX, e.clientY);
                this.drawTempWire(this.draggingWire.startX, this.draggingWire.startY, mousePos.x, mousePos.y);
            }
        });
        
        // Mouse up ends any active drag
        window.addEventListener('mouseup', (e) => {
            if (this.isPanning) {
                this.isPanning = false;
                this.container.style.cursor = this.isSpacePressed ? 'grab' : 'default';
            }
            
            if (this.draggedNode) {
                this.draggedNode = null;
            }
            
            if (this.draggingWire) {
                // Check if dropped over a valid port
                const target = e.target;
                if (target.classList.contains('port') && target.dataset.portType === 'input') {
                    const toNodeId = target.dataset.nodeId;
                    const toPort = target.dataset.portName;
                    
                    this.createConnection(
                        this.draggingWire.fromNode,
                        this.draggingWire.fromPort,
                        toNodeId,
                        toPort
                    );
                }
                
                this.draggingWire = null;
                this.tempWire.style.display = 'none';
            }
        });
        
        // Zoom via mouse wheel
        this.container.addEventListener('wheel', (e) => {
            e.preventDefault();
            const zoomFactor = 0.05;
            const prevZoom = this.zoom;
            
            if (e.deltaY < 0) {
                this.zoom = Math.min(this.zoom + zoomFactor, 2.0);
            } else {
                this.zoom = Math.max(this.zoom - zoomFactor, 0.4);
            }
            
            // Zoom centered on mouse position
            const mousePos = this.container.getBoundingClientRect();
            const mx = e.clientX - mousePos.left;
            const my = e.clientY - mousePos.top;
            
            this.panX = mx - (mx - this.panX) * (this.zoom / prevZoom);
            this.panY = my - (my - this.panY) * (this.zoom / prevZoom);
            
            this.updateCanvasTransform();
        });
        
        // Deselect node on canvas click
        this.container.addEventListener('click', (e) => {
            if (e.target === this.container || e.target === this.svg) {
                this.deselectAll();
            }
        });
    }
    
    // Convert screen pixel coordinates to canvas coordinate space
    getCanvasCoordinates(clientX, clientY) {
        const rect = this.nodesContainer.getBoundingClientRect();
        return {
            x: (clientX - rect.left) / this.zoom,
            y: (clientY - rect.top) / this.zoom
        };
    }
    
    updateCanvasTransform() {
        const el = document.getElementById('canvas-content');
        el.style.transform = `translate(${this.panX}px, ${this.panY}px) scale(${this.zoom})`;
    }
    
    deselectAll() {
        if (this.selectedNode) {
            const el = document.getElementById(this.selectedNode.id);
            if (el) el.classList.remove('selected');
            this.selectedNode = null;
        }
    }
    
    selectNode(node) {
        this.deselectAll();
        this.selectedNode = node;
        const el = document.getElementById(node.id);
        if (el) el.classList.add('selected');
    }
    
    // Add a node to the designer
    addNode(type, name, x = 100, y = 100, id = null) {
        const nodeId = id || `node_${Math.random().toString(36).substr(2, 9)}`;
        const node = {
            id: nodeId,
            type: type,
            name: name,
            x: x,
            y: y,
            fileInputId: '', // For file uploads
            fileInputName: ''
        };
        
        this.nodes.push(node);
        this.renderNodeHTML(node);
        return node;
    }
    
    deleteNode(nodeId) {
        // Remove connections
        this.connections = this.connections.filter(conn => {
            return conn.fromNode !== nodeId && conn.toNode !== nodeId;
        });
        
        // Remove node from state
        this.nodes = this.nodes.filter(n => n.id !== nodeId);
        
        // Remove from DOM
        const el = document.getElementById(nodeId);
        if (el) el.remove();
        
        this.drawAllConnections();
        
        if (this.selectedNode && this.selectedNode.id === nodeId) {
            this.selectedNode = null;
        }
    }
    
    renderNodeHTML(node) {
        const nodeEl = document.createElement('div');
        nodeEl.id = node.id;
        nodeEl.className = `node node-${node.type.replace('.', '-')}`;
        nodeEl.style.left = `${node.x}px`;
        nodeEl.style.top = `${node.y}px`;
        
        // Header
        const header = document.createElement('div');
        header.className = 'node-header';
        
        const titleSpan = document.createElement('span');
        titleSpan.textContent = node.name;
        titleSpan.contentEditable = true;
        titleSpan.className = 'node-title-editable';
        titleSpan.addEventListener('blur', () => {
            node.name = titleSpan.textContent.trim() || node.type;
        });
        
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'btn-delete';
        deleteBtn.innerHTML = '&times;';
        deleteBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            this.deleteNode(node.id);
        });
        
        header.appendChild(titleSpan);
        header.appendChild(deleteBtn);
        
        // Body
        const body = document.createElement('div');
        body.className = 'node-body';
        
        const badge = document.createElement('span');
        badge.className = 'node-type-badge';
        badge.textContent = node.type;
        body.appendChild(badge);
        
        // Custom Controls based on task type
        if (node.type === 'csv.read' || node.type === 'doc.read') {
            const uploadWrapper = document.createElement('div');
            uploadWrapper.className = 'file-upload-wrapper';
            
            const fileInput = document.createElement('input');
            fileInput.type = 'file';
            fileInput.className = 'file-input-hidden';
            fileInput.accept = node.type === 'csv.read' ? '.csv' : '.txt,.doc,.docx,.pdf';
            
            const uploadBtn = document.createElement('div');
            uploadBtn.className = 'file-upload-btn';
            uploadBtn.textContent = 'Upload File';
            
            const statusText = document.createElement('div');
            statusText.className = 'file-status-text';
            statusText.textContent = node.fileInputName || 'No file selected';
            
            uploadBtn.addEventListener('click', () => fileInput.click());
            fileInput.addEventListener('change', async (e) => {
                const file = e.target.files[0];
                if (!file) return;
                
                statusText.textContent = `Uploading: ${file.name}...`;
                try {
                    const minioFile = await TaskTideAPI.uploadFile(file);
                    node.fileInputId = minioFile.id;
                    node.fileInputName = file.name;
                    statusText.textContent = `Uploaded: ${file.name}`;
                    statusText.title = `ID: ${minioFile.id}`;
                    this.showStatus('File uploaded successfully', 'success');
                } catch (err) {
                    console.error(err);
                    statusText.textContent = 'Upload failed!';
                    this.showStatus('File upload failed', 'error');
                }
            });
            
            uploadWrapper.appendChild(fileInput);
            uploadWrapper.appendChild(uploadBtn);
            uploadWrapper.appendChild(statusText);
            body.appendChild(uploadWrapper);
        }
        
        // Ports Footers
        const ports = document.createElement('div');
        ports.className = 'node-ports';
        
        // Inputs: output node or general tasks have input ports
        if (node.type !== 'csv.read' && node.type !== 'doc.read') {
            const inputContainer = document.createElement('div');
            inputContainer.className = 'port-container input';
            
            const port = document.createElement('div');
            port.className = 'port';
            port.dataset.nodeId = node.id;
            port.dataset.portType = 'input';
            port.dataset.portName = 'input';
            
            const label = document.createElement('span');
            label.className = 'port-label';
            label.textContent = 'Input';
            
            inputContainer.appendChild(port);
            inputContainer.appendChild(label);
            ports.appendChild(inputContainer);
        }
        
        // Outputs: output node does not have an output port
        if (node.type !== 'OUTPUT') {
            const outputContainer = document.createElement('div');
            outputContainer.className = 'port-container output';
            
            const port = document.createElement('div');
            port.className = 'port';
            port.dataset.nodeId = node.id;
            port.dataset.portType = 'output';
            port.dataset.portName = 'output';
            
            // Wire drag starting from output port
            port.addEventListener('mousedown', (e) => {
                e.stopPropagation();
                e.preventDefault();
                const portCenter = this.getPortCoordinates(node.id, 'output');
                this.draggingWire = {
                    fromNode: node.id,
                    fromPort: 'output',
                    startX: portCenter.x,
                    startY: portCenter.y
                };
                this.tempWire.style.display = 'block';
            });
            
            const label = document.createElement('span');
            label.className = 'port-label';
            label.textContent = 'Output';
            
            outputContainer.appendChild(port);
            outputContainer.appendChild(label);
            ports.appendChild(outputContainer);
        }
        
        nodeEl.appendChild(header);
        nodeEl.appendChild(body);
        nodeEl.appendChild(ports);
        
        // Node dragging events
        header.addEventListener('mousedown', (e) => {
            if (e.button !== 0 || this.isSpacePressed) return;
            e.stopPropagation();
            this.draggedNode = node;
            this.selectNode(node);
            
            const rect = nodeEl.getBoundingClientRect();
            this.nodeOffsetX = e.clientX - rect.left;
            this.nodeOffsetY = e.clientY - rect.top;
        });
        
        nodeEl.addEventListener('click', (e) => {
            e.stopPropagation();
            this.selectNode(node);
        });
        
        this.nodesContainer.appendChild(nodeEl);
    }
    
    updateNodePosition(node) {
        const el = document.getElementById(node.id);
        if (el) {
            el.style.left = `${node.x}px`;
            el.style.top = `${node.y}px`;
        }
    }
    
    // Find output or input port absolute coordinates on the canvas
    getPortCoordinates(nodeId, portType) {
        const nodeEl = document.getElementById(nodeId);
        if (!nodeEl) return { x: 0, y: 0 };
        
        const portEl = nodeEl.querySelector(`.port[data-port-type="${portType}"]`);
        if (!portEl) return { x: 0, y: 0 };
        
        const rect = portEl.getBoundingClientRect();
        const canvasRect = this.nodesContainer.getBoundingClientRect();
        
        return {
            x: (rect.left + rect.width / 2 - canvasRect.left) / this.zoom,
            y: (rect.top + rect.height / 2 - canvasRect.top) / this.zoom
        };
    }
    
    // Connect output port of fromNode to input port of toNode
    createConnection(fromNodeId, fromPort, toNodeId, toPort) {
        // Verify no cycle
        if (this.wouldCreateCycle(toNodeId, fromNodeId)) {
            this.showStatus('Cannot connect: creates a cyclic dependency', 'error');
            return;
        }
        
        // Remove existing connection to target input port (each task takes at most one input task in TaskTide)
        this.connections = this.connections.filter(conn => !(conn.toNode === toNodeId && conn.toPort === toPort));
        
        // Update port visual class state
        const connId = `conn_${Math.random().toString(36).substr(2, 9)}`;
        this.connections.push({
            id: connId,
            fromNode: fromNodeId,
            fromPort: fromPort,
            toNode: toNodeId,
            toPort: toPort
        });
        
        this.drawAllConnections();
        this.updatePortsVisualState();
    }
    
    updatePortsVisualState() {
        // Mark ports active if connected
        document.querySelectorAll('.port').forEach(p => p.classList.remove('connected'));
        
        this.connections.forEach(conn => {
            const fromNodeEl = document.getElementById(conn.fromNode);
            const toNodeEl = document.getElementById(conn.toNode);
            
            if (fromNodeEl) {
                const port = fromNodeEl.querySelector(`.port[data-port-type="output"]`);
                if (port) port.classList.add('connected');
            }
            if (toNodeEl) {
                const port = toNodeEl.querySelector(`.port[data-port-type="input"]`);
                if (port) port.classList.add('connected');
            }
        });
    }
    
    wouldCreateCycle(startNodeId, targetNodeId) {
        if (startNodeId === targetNodeId) return true;
        
        // Simple BFS to check reachability
        const visited = new Set();
        const queue = [startNodeId];
        
        while (queue.length > 0) {
            const curr = queue.shift();
            if (curr === targetNodeId) return true;
            
            visited.add(curr);
            
            // Get all children (nodes where curr outputs to)
            const children = this.connections
                .filter(conn => conn.fromNode === curr)
                .map(conn => conn.toNode);
                
            for (const child of children) {
                if (!visited.has(child)) {
                    queue.push(child);
                }
            }
        }
        return false;
    }
    
    // Draw all wires connecting nodes
    drawAllConnections() {
        this.connectionsGroup.innerHTML = '';
        
        this.connections.forEach(conn => {
            const start = this.getPortCoordinates(conn.fromNode, 'output');
            const end = this.getPortCoordinates(conn.toNode, 'input');
            
            const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            const dx = Math.abs(end.x - start.x) * 0.5;
            
            // Smooth cubic Bezier wire
            const d = `M ${start.x} ${start.y} C ${start.x + dx} ${start.y}, ${end.x - dx} ${end.y}, ${end.x} ${end.y}`;
            
            path.setAttribute('d', d);
            path.setAttribute('stroke', 'url(#wire-gradient)');
            path.setAttribute('stroke-width', '3');
            path.setAttribute('filter', 'url(#wire-glow)');
            path.style.cursor = 'pointer';
            path.style.pointerEvents = 'auto'; // allow clicking wires to delete them
            
            // Wire right click deletes it
            path.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                e.stopPropagation();
                this.deleteConnection(conn.id);
            });
            
            this.connectionsGroup.appendChild(path);
        });
        
        this.updatePortsVisualState();
    }
    
    deleteConnection(connId) {
        this.connections = this.connections.filter(c => c.id !== connId);
        this.drawAllConnections();
    }
    
    drawTempWire(x1, y1, x2, y2) {
        const dx = Math.abs(x2 - x1) * 0.5;
        const d = `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
        this.tempWire.setAttribute('d', d);
    }
    
    // Clear whole workspace
    clear() {
        this.nodes = [];
        this.connections = [];
        this.nodesContainer.innerHTML = '';
        this.connectionsGroup.innerHTML = '';
        this.selectedNode = null;
        this.updateCanvasTransform();
    }
    
    // Topological Sort nodes so they can be saved sequentially to satisfy backend dependencies
    topologicalSort() {
        const sorted = [];
        const visited = new Set();
        const temp = new Set();
        
        const visit = (nodeId) => {
            if (temp.has(nodeId)) {
                throw new Error('Cycle detected');
            }
            if (!visited.has(nodeId)) {
                temp.add(nodeId);
                
                // Find node dependency (nodes feeding into nodeId input)
                const dependencies = this.connections
                    .filter(conn => conn.toNode === nodeId)
                    .map(conn => conn.fromNode);
                    
                for (const depId of dependencies) {
                    visit(depId);
                }
                
                temp.delete(nodeId);
                visited.add(nodeId);
                
                const node = this.nodes.find(n => n.id === nodeId);
                if (node) sorted.push(node);
            }
        };
        
        for (const node of this.nodes) {
            visit(node.id);
        }
        
        return sorted;
    }
    
    // Saves graph to TaskTide backend
    async saveWorkflow(workflowName) {
        try {
            this.showStatus('Saving workflow...', 'running');
            
            // 1. Create Workflow Definition
            const workflow = await TaskTideAPI.createWorkflow(workflowName);
            const workflowId = workflow.id;
            
            // 2. Sort nodes topologically
            const sortedNodes = this.topologicalSort();
            const clientToBackendMap = {}; // Maps client node ID to saved Go backend task ID
            
            // 3. Save tasks sequentially
            for (const node of sortedNodes) {
                // Determine task input task ID if connected
                let inputTaskId = '';
                const incomingConn = this.connections.find(conn => conn.toNode === node.id);
                if (incomingConn) {
                    inputTaskId = clientToBackendMap[incomingConn.fromNode] || '';
                }
                
                const taskPayload = {
                    task_type: node.type,
                    task_name: node.name,
                    workflow_id: workflowId,
                    task_input: node.fileInputId || '',
                    task_input_task: inputTaskId
                };
                
                const createdTask = await TaskTideAPI.addTask(taskPayload);
                clientToBackendMap[node.id] = createdTask.id;
            }
            
            // 4. Update workflow definition output file if we have an output node
            const outputNode = this.nodes.find(n => n.type === 'OUTPUT');
            if (outputNode) {
                // If output node takes input from another task, set workflow output
                const incomingOutputConn = this.connections.find(conn => conn.toNode === outputNode.id);
                if (incomingOutputConn) {
                    const finalTaskId = clientToBackendMap[incomingOutputConn.fromNode];
                    // We can map the dependency
                }
            }
            
            this.showStatus('Workflow saved successfully!', 'success');
            return workflowId;
        } catch (err) {
            console.error(err);
            this.showStatus(`Save failed: ${err.message}`, 'error');
            throw err;
        }
    }
    
    // Load a fat workflow definition and render as a node graph
    async loadWorkflow(workflowId) {
        this.clear();
        this.showStatus('Loading workflow...', 'running');
        
        try {
            const fatWorkflow = await TaskTideAPI.getWorkflow(workflowId);
            const tasks = fatWorkflow.workflow_tasks || [];
            
            // Draw nodes in grid/horizontal layout
            const backendToClientMap = {};
            let col = 0;
            let row = 0;
            
            // Map tasks to draw them
            tasks.forEach((task, idx) => {
                const x = 100 + col * 300;
                const y = 150 + row * 180;
                
                // Add the node
                const node = this.addNode(task.task_type, task.task_name, x, y, task.id);
                backendToClientMap[task.id] = task.id;
                
                // Set file input if it exists
                if (task.task_input) {
                    node.fileInputId = task.task_input;
                    node.fileInputName = `Attached File (${task.task_input.substr(0,8)})`;
                    const statusText = document.getElementById(task.id).querySelector('.file-status-text');
                    if (statusText) {
                        statusText.textContent = node.fileInputName;
                        statusText.title = `ID: ${task.task_input}`;
                    }
                }
                
                col++;
                if (col > 3) {
                    col = 0;
                    row++;
                }
            });
            
            // Draw an output node at the end if tasks don't already have one
            const hasOutputNode = tasks.some(t => t.task_type === 'OUTPUT');
            let outputNodeId = '';
            if (!hasOutputNode && tasks.length > 0) {
                const outNode = this.addNode('OUTPUT', 'Workflow Output', 100 + col * 300, 150 + row * 180);
                outputNodeId = outNode.id;
            }
            
            // Establish connection wires
            tasks.forEach(task => {
                if (task.task_input_task) {
                    this.createConnection(
                        task.task_input_task,
                        'output',
                        task.id,
                        'input'
                    );
                }
                
                // If it's an output task type or if this task feeds into a missing output node
                if (task.task_type === 'OUTPUT' && task.task_input_task) {
                    this.createConnection(task.task_input_task, 'output', task.id, 'input');
                }
            });
            
            // Connect last task to the auto-created output node if applicable
            if (outputNodeId && tasks.length > 0) {
                // Find the task that has no children nodes depending on it
                const leafTasks = tasks.filter(t => {
                    return !tasks.some(other => other.task_input_task === t.id);
                });
                
                if (leafTasks.length > 0) {
                    this.createConnection(leafTasks[0].id, 'output', outputNodeId, 'input');
                }
            }
            
            this.showStatus('Workflow loaded successfully', 'success');
        } catch (err) {
            console.error(err);
            this.showStatus('Failed to load workflow', 'error');
        }
    }
    
    // Updates node execution highlights during run monitor
    updateTaskStates(taskExecutions) {
        // Clear all current execution status highlights on nodes
        this.nodes.forEach(node => {
            const el = document.getElementById(node.id);
            if (el) {
                el.classList.remove('state-pending', 'state-running', 'state-completed', 'state-failed');
            }
        });
        
        taskExecutions.forEach(exec => {
            // exec.task_base is the task ID
            const el = document.getElementById(exec.task_base);
            if (el) {
                if (exec.task_completed) {
                    el.classList.add('state-completed');
                } else if (exec.task_assigned) {
                    el.classList.add('state-running');
                } else {
                    el.classList.add('state-pending');
                }
            }
        });
    }
    
    // Toast Status Logger
    showStatus(msg, type = 'info') {
        const dot = document.querySelector('.status-dot');
        const text = document.getElementById('status-text');
        
        dot.className = 'status-dot';
        if (type === 'success') dot.classList.add('active');
        if (type === 'running') dot.classList.add('running');
        if (type === 'error') {
            dot.style.backgroundColor = 'var(--accent-danger)';
            dot.style.boxShadow = '0 0 8px var(--accent-danger)';
        }
        
        text.textContent = msg;
    }
}
