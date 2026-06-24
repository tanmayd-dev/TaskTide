// api.js - TaskTide Backend API Service Client

const API_BASE = 'http://localhost:8080';

export const TaskTideAPI = {
    // List all workflow definitions
    async listWorkflows() {
        const res = await fetch(`${API_BASE}/workflow/list`);
        if (!res.ok) throw new Error('Failed to fetch workflows list');
        return res.json();
    },

    // Get a specific fat workflow definition by ID (includes its tasks)
    async getWorkflow(id) {
        const res = await fetch(`${API_BASE}/workflow/${id}`);
        if (!res.ok) throw new Error(`Failed to fetch workflow ${id}`);
        return res.json();
    },

    // Create a new workflow definition
    async createWorkflow(name) {
        const res = await fetch(`${API_BASE}/workflow/create`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ workflow_name: name })
        });
        if (!res.ok) throw new Error('Failed to create workflow');
        return res.json();
    },

    // Add a task definition to a workflow
    async addTask(task) {
        const res = await fetch(`${API_BASE}/workflow/add`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(task)
        });
        if (!res.ok) throw new Error('Failed to add task to workflow');
        return res.json();
    },

    // Upload an input file (binds to a reader node)
    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);
        
        const res = await fetch(`${API_BASE}/file`, {
            method: 'POST',
            body: formData // Content-Type is set automatically by the browser with boundary
        });
        if (!res.ok) throw new Error('Failed to upload file');
        return res.json(); // Returns models.InputFile { id, file_name }
    },

    // Start a workflow execution run
    async startWorkflow(workflowId) {
        const res = await fetch(`${API_BASE}/workflow/`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: workflowId })
        });
        if (!res.ok) throw new Error('Failed to start workflow execution');
        return res.json(); // Returns { id: executionID }
    },

    // Get current task execution statuses for a workflow run
    async getWorkflowStatus(executionId) {
        const res = await fetch(`${API_BASE}/workflow/status`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: executionId })
        });
        if (!res.ok) throw new Error('Failed to fetch workflow execution status');
        return res.json(); // Returns []models.TaskExecution
    },

    // Download a file by ID using query parameter
    async downloadFile(fileId, fileName) {
        try {
            const res = await fetch(`${API_BASE}/file?id=${fileId}`);
            if (!res.ok) throw new Error('Failed to download file content');
            
            const blob = await res.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = fileName || `file_${fileId.substring(0, 8)}`;
            document.body.appendChild(a);
            a.click();
            
            // Cleanup
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
        } catch (err) {
            console.error('File download failed:', err);
            alert(`Download failed: ${err.message}`);
        }
    }
};
