export interface Task {
  id: string;
  title: string;
  description?: string;
  status: TaskStatus;
  priority: Priority;
  assigneeId?: string;
  assigneeName?: string;
  estimatedHours?: number;
  actualHours?: number;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  position: number;
  columnId: string;
  boardId: string;
  riskLevel?: RiskLevel;
  velocityPrediction?: VelocityPrediction;
  dependencies?: string[];
  blockers?: string[];
}

export interface Column {
  id: string;
  title: string;
  status: TaskStatus;
  position: number;
  boardId: string;
  wipLimit?: number;
  tasks: Task[];
  riskLevel?: RiskLevel;
}

export interface Board {
  id: string;
  name: string;
  description?: string;
  projectId: string;
  columns: Column[];
  teamMembers: TeamMember[];
  createdAt: string;
  updatedAt: string;
}

export interface TeamMember {
  id: string;
  name: string;
  email: string;
  avatar?: string;
  role: Role;
  capacity: number; // hours per sprint
  currentLoad: number; // current assigned hours
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  boards: Board[];
  teamMembers: TeamMember[];
  sprintDuration: number; // days
  createdAt: string;
  updatedAt: string;
}

export type TaskStatus = 'todo' | 'in-progress' | 'review' | 'done';
export type Priority = 'low' | 'medium' | 'high' | 'urgent';
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical';
export type Role = 'developer' | 'designer' | 'qa' | 'manager' | 'product-owner';

export interface VelocityPrediction {
  estimatedCompletion: string;
  confidence: number; // 0-1
  factors: PredictionFactor[];
  riskLevel: RiskLevel;
  suggestedActions?: string[];
}

export interface PredictionFactor {
  name: string;
  impact: number; // -1 to 1
  description: string;
}

export interface TaskEvent {
  id: string;
  type: EventType;
  taskId: string;
  boardId: string;
  userId: string;
  timestamp: string;
  data: Record<string, any>;
  previousState?: Partial<Task>;
  newState?: Partial<Task>;
}

export type EventType = 
  | 'task-created'
  | 'task-updated'
  | 'task-moved'
  | 'task-assigned'
  | 'task-completed'
  | 'task-deleted'
  | 'risk-alert'
  | 'reallocation-suggested'
  | 'reallocation-approved'
  | 'reallocation-rejected';

export interface RiskAlert {
  id: string;
  taskId: string;
  boardId: string;
  riskLevel: RiskLevel;
  message: string;
  factors: PredictionFactor[];
  timestamp: string;
  acknowledged: boolean;
  suggestedActions: ReallocationSuggestion[];
}

export interface ReallocationSuggestion {
  id: string;
  taskId: string;
  fromUserId?: string;
  toUserId: string;
  reasoning: string;
  impact: {
    velocityImprovement: number;
    riskReduction: number;
    loadBalance: number;
  };
  confidence: number;
  timestamp: string;
  status: 'pending' | 'approved' | 'rejected';
  approvedBy?: string;
  approvedAt?: string;
}

export interface AgentAction {
  id: string;
  type: 'reallocation' | 'priority-change' | 'deadline-extension';
  description: string;
  confidence: number;
  impact: AgentImpact;
  status: 'pending' | 'approved' | 'rejected' | 'executed';
  createdAt: string;
  executedAt?: string;
  approvedBy?: string;
}

export interface AgentImpact {
  velocityChange: number;
  riskChange: number;
  teamLoadChange: number;
  estimatedBenefit: string;
}

export interface DragState {
  isDragging: boolean;
  draggedTask?: Task;
  sourceColumn?: string;
  targetColumn?: string;
  dragPosition?: { x: number; y: number };
}

export interface VirtualScrollConfig {
  itemHeight: number;
  overscan: number;
  containerHeight: number;
  threshold: number;
}

export interface AnalyticsData {
  velocity: VelocityMetric[];
  burndown: BurndownData[];
  riskTrends: RiskTrendData[];
  teamPerformance: TeamPerformanceData[];
  predictionAccuracy: PredictionAccuracyData[];
}

export interface VelocityMetric {
  date: string;
  completed: number;
  planned: number;
  actual: number;
  prediction: number;
}

export interface BurndownData {
  date: string;
  remaining: number;
  ideal: number;
  actual: number;
}

export interface RiskTrendData {
  date: string;
  low: number;
  medium: number;
  high: number;
  critical: number;
}

export interface TeamPerformanceData {
  memberId: string;
  memberName: string;
  completed: number;
  capacity: number;
  efficiency: number;
  riskLevel: RiskLevel;
}

export interface PredictionAccuracyData {
  date: string;
  predicted: number;
  actual: number;
  accuracy: number;
  confidence: number;
}
