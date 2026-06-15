// api.js - TaskTide Backend API Service Client

const API_BASE = 'http://localhost:8080';

export const TaskTideAPI = {
    // List all workflow definitions
    async listWorkflows() {
        const res = await fetch(`${API_BASE}/workflow/list`);
        if (!res.ok) throw new Error('Failed to fetch workflows list');
        return res.json();
    },

    // Get a specific fat workflow definition by ID
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

    // Add task dependency details
    async addTaskDependency(id, dependencies, inputFromDependency) {
        const res = await fetch(`${API_BASE}/workflow/dependency`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                id: id,
                dependencies: dependencies,
                input_from_dependency: inputFromDependency
            })
        });
        if (!res.ok) throw new Error('Failed to add task dependency');
        return res;
    },

    // Upload an input file
    async uploadFile(file) {
        const formData = new FormData();
        formData.append('file', file);
        
        const res = await fetch(`${API_BASE}/file`, {
            method: 'POST',
            body: formData // Content-Type is set automatically by the browser with boundary
        });
        if (!res.ok) throw new Error('Failed to upload file');
        return res.json();
    },

    // Start a workflow execution
    async startWorkflow(workflowId) {
        const res = await fetch(`${API_BASE}/workflow/`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: workflowId })
        });
        if (!res.ok) throw new Error('Failed to start workflow execution');
        return res.json(); // Returns { id: executionID }
    },

    // Get current task execution statuses for a workflow execution run
    async getWorkflowStatus(executionId) {
        const res = await fetch(`${API_BASE}/workflow/status`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ id: executionId })
        });
        if (!res.ok) throw new Error('Failed to fetch workflow execution status');
        return res.json(); // Returns []models.TaskExecution
    }
};
