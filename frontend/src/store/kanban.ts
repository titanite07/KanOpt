import { create } from 'zustand';
import { subscribeWithSelector } from 'zustand/middleware';
import { Task, Column, Board, TeamMember, TaskEvent, RiskAlert, ReallocationSuggestion, DragState, AgentAction } from '@/types';

interface KanbanState {
  // Data
  boards: Board[];
  currentBoard: Board | null;
  teamMembers: TeamMember[];
  events: TaskEvent[];
  riskAlerts: RiskAlert[];
  suggestions: ReallocationSuggestion[];
  agentActions: AgentAction[];
  
  // UI State
  dragState: DragState;
  isLoading: boolean;
  error: string | null;
  
  // View State
  showRiskOverlay: boolean;
  showVelocityPredictions: boolean;
  selectedTaskId: string | null;
  filterTags: string[];
  searchQuery: string;
  
  // Actions
  setBoards: (boards: Board[]) => void;
  setCurrentBoard: (board: Board | null) => void;
  addTask: (task: Task) => void;
  updateTask: (taskId: string, updates: Partial<Task>) => void;
  moveTask: (taskId: string, sourceColumnId: string, targetColumnId: string, position: number) => void;
  deleteTask: (taskId: string) => void;
  
  setDragState: (state: Partial<DragState>) => void;
  clearDragState: () => void;
  
  addEvent: (event: TaskEvent) => void;
  addRiskAlert: (alert: RiskAlert) => void;
  acknowledgeRiskAlert: (alertId: string) => void;
  
  addSuggestion: (suggestion: ReallocationSuggestion) => void;
  approveSuggestion: (suggestionId: string) => void;
  rejectSuggestion: (suggestionId: string) => void;
  
  addAgentAction: (action: AgentAction) => void;
  approveAgentAction: (actionId: string) => void;
  rejectAgentAction: (actionId: string) => void;
  
  setShowRiskOverlay: (show: boolean) => void;
  setShowVelocityPredictions: (show: boolean) => void;
  setSelectedTask: (taskId: string | null) => void;
  setFilterTags: (tags: string[]) => void;
  setSearchQuery: (query: string) => void;
  
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  
  // Computed getters
  getCurrentTasks: () => Task[];
  getTasksByColumn: (columnId: string) => Task[];
  getFilteredTasks: () => Task[];
  getRiskAlertsByLevel: (level: string) => RiskAlert[];
  getPendingSuggestions: () => ReallocationSuggestion[];
  getPendingAgentActions: () => AgentAction[];
}

export const useKanbanStore = create<KanbanState>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    boards: [],
    currentBoard: null,
    teamMembers: [],
    events: [],
    riskAlerts: [],
    suggestions: [],
    agentActions: [],
    
    dragState: {
      isDragging: false,
    },
    isLoading: false,
    error: null,
    
    showRiskOverlay: false,
    showVelocityPredictions: true,
    selectedTaskId: null,
    filterTags: [],
    searchQuery: '',
    
    // Actions
    setBoards: (boards) => set({ boards }),
    
    setCurrentBoard: (board) => set({ currentBoard: board }),
    
    addTask: (task) => set((state) => {
      if (!state.currentBoard) return state;
      
      const updatedColumns = state.currentBoard.columns.map(column => 
        column.id === task.columnId 
          ? { ...column, tasks: [...column.tasks, task] }
          : column
      );
      
      const updatedBoard = { ...state.currentBoard, columns: updatedColumns };
      const updatedBoards = state.boards.map(b => 
        b.id === updatedBoard.id ? updatedBoard : b
      );
      
      return {
        boards: updatedBoards,
        currentBoard: updatedBoard,
      };
    }),
    
    updateTask: (taskId, updates) => set((state) => {
      if (!state.currentBoard) return state;
      
      const updatedColumns = state.currentBoard.columns.map(column => ({
        ...column,
        tasks: column.tasks.map(task => 
          task.id === taskId ? { ...task, ...updates } : task
        ),
      }));
      
      const updatedBoard = { ...state.currentBoard, columns: updatedColumns };
      const updatedBoards = state.boards.map(b => 
        b.id === updatedBoard.id ? updatedBoard : b
      );
      
      return {
        boards: updatedBoards,
        currentBoard: updatedBoard,
      };
    }),
    
    moveTask: (taskId, sourceColumnId, targetColumnId, position) => set((state) => {
      if (!state.currentBoard) return state;
      
      let movedTask: Task | null = null;
      
      // Remove task from source column
      const updatedColumns = state.currentBoard.columns.map(column => {
        if (column.id === sourceColumnId) {
          const taskIndex = column.tasks.findIndex(t => t.id === taskId);
          if (taskIndex !== -1) {
            movedTask = { ...column.tasks[taskIndex], columnId: targetColumnId };
            return {
              ...column,
              tasks: column.tasks.filter(t => t.id !== taskId),
            };
          }
        }
        return column;
      });
      
      // Add task to target column
      if (movedTask) {
        const finalColumns = updatedColumns.map(column => {
          if (column.id === targetColumnId) {
            const newTasks = [...column.tasks];
            newTasks.splice(position, 0, movedTask!);
            return { ...column, tasks: newTasks };
          }
          return column;
        });
        
        const updatedBoard = { ...state.currentBoard, columns: finalColumns };
        const updatedBoards = state.boards.map(b => 
          b.id === updatedBoard.id ? updatedBoard : b
        );
        
        return {
          boards: updatedBoards,
          currentBoard: updatedBoard,
        };
      }
      
      return state;
    }),
    
    deleteTask: (taskId) => set((state) => {
      if (!state.currentBoard) return state;
      
      const updatedColumns = state.currentBoard.columns.map(column => ({
        ...column,
        tasks: column.tasks.filter(task => task.id !== taskId),
      }));
      
      const updatedBoard = { ...state.currentBoard, columns: updatedColumns };
      const updatedBoards = state.boards.map(b => 
        b.id === updatedBoard.id ? updatedBoard : b
      );
      
      return {
        boards: updatedBoards,
        currentBoard: updatedBoard,
      };
    }),
    
    setDragState: (dragState) => set((state) => ({
      dragState: { ...state.dragState, ...dragState }
    })),
    
    clearDragState: () => set({
      dragState: { isDragging: false }
    }),
    
    addEvent: (event) => set((state) => ({
      events: [event, ...state.events]
    })),
    
    addRiskAlert: (alert) => set((state) => ({
      riskAlerts: [alert, ...state.riskAlerts]
    })),
    
    acknowledgeRiskAlert: (alertId) => set((state) => ({
      riskAlerts: state.riskAlerts.map(alert =>
        alert.id === alertId ? { ...alert, acknowledged: true } : alert
      )
    })),
    
    addSuggestion: (suggestion) => set((state) => ({
      suggestions: [suggestion, ...state.suggestions]
    })),
    
    approveSuggestion: (suggestionId) => set((state) => ({
      suggestions: state.suggestions.map(suggestion =>
        suggestion.id === suggestionId 
          ? { ...suggestion, status: 'approved', approvedAt: new Date().toISOString() }
          : suggestion
      )
    })),
    
    rejectSuggestion: (suggestionId) => set((state) => ({
      suggestions: state.suggestions.map(suggestion =>
        suggestion.id === suggestionId 
          ? { ...suggestion, status: 'rejected' }
          : suggestion
      )
    })),
    
    addAgentAction: (action) => set((state) => ({
      agentActions: [action, ...state.agentActions]
    })),
    
    approveAgentAction: (actionId) => set((state) => ({
      agentActions: state.agentActions.map(action =>
        action.id === actionId 
          ? { ...action, status: 'approved', executedAt: new Date().toISOString() }
          : action
      )
    })),
    
    rejectAgentAction: (actionId) => set((state) => ({
      agentActions: state.agentActions.map(action =>
        action.id === actionId 
          ? { ...action, status: 'rejected' }
          : action
      )
    })),
    
    setShowRiskOverlay: (show) => set({ showRiskOverlay: show }),
    setShowVelocityPredictions: (show) => set({ showVelocityPredictions: show }),
    setSelectedTask: (taskId) => set({ selectedTaskId: taskId }),
    setFilterTags: (tags) => set({ filterTags: tags }),
    setSearchQuery: (query) => set({ searchQuery: query }),
    setLoading: (loading) => set({ isLoading: loading }),
    setError: (error) => set({ error }),
    
    // Computed getters
    getCurrentTasks: () => {
      const state = get();
      if (!state.currentBoard) return [];
      return state.currentBoard.columns.flatMap(column => column.tasks);
    },
    
    getTasksByColumn: (columnId) => {
      const state = get();
      if (!state.currentBoard) return [];
      const column = state.currentBoard.columns.find(c => c.id === columnId);
      return column?.tasks || [];
    },
    
    getFilteredTasks: () => {
      const state = get();
      const tasks = state.getCurrentTasks();
      let filtered = tasks;
      
      if (state.searchQuery) {
        filtered = filtered.filter(task => 
          task.title.toLowerCase().includes(state.searchQuery.toLowerCase()) ||
          task.description?.toLowerCase().includes(state.searchQuery.toLowerCase())
        );
      }
      
      if (state.filterTags.length > 0) {
        filtered = filtered.filter(task =>
          state.filterTags.some(tag => task.tags.includes(tag))
        );
      }
      
      return filtered;
    },
    
    getRiskAlertsByLevel: (level) => {
      const state = get();
      return state.riskAlerts.filter(alert => alert.riskLevel === level);
    },
    
    getPendingSuggestions: () => {
      const state = get();
      return state.suggestions.filter(suggestion => suggestion.status === 'pending');
    },
    
    getPendingAgentActions: () => {
      const state = get();
      return state.agentActions.filter(action => action.status === 'pending');
    },
  }))
);
