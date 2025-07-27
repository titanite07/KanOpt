'use client';

import { useEffect, useState } from 'react';
import KanbanBoard from '@/components/KanbanBoard';
import AgentPanel from '@/components/AgentPanel';
import AnalyticsDashboard from '@/components/AnalyticsDashboard';
import { useKanbanStore } from '@/store/kanban';
import { Board, Task, Column } from '@/types';

const mockBoard: Board = {
  id: 'board-1',
  name: 'Sprint 24.1 - AI Features',
  description: 'Implementing AI-powered features for the platform',
  projectId: 'project-1',
  columns: [
    {
      id: 'col-todo',
      title: 'Backlog',
      status: 'todo',
      position: 0,
      boardId: 'board-1',
      wipLimit: 10,
      tasks: [
        {
          id: 'task-1',
          title: 'Implement LSTM velocity predictor',
          description: 'Build neural network for predicting task completion times',
          status: 'todo',
          priority: 'high',
          assigneeId: 'user-1',
          assigneeName: 'Alice Johnson',
          estimatedHours: 16,
          tags: ['AI', 'ML', 'Backend'],
          createdAt: '2025-01-20T09:00:00Z',
          updatedAt: '2025-01-20T09:00:00Z',
          position: 0,
          columnId: 'col-todo',
          boardId: 'board-1',
          riskLevel: 'medium',
          velocityPrediction: {
            estimatedCompletion: '2025-01-25T17:00:00Z',
            confidence: 0.85,
            factors: [
              { name: 'Complexity', impact: -0.3, description: 'High algorithmic complexity' },
              { name: 'Experience', impact: 0.4, description: 'Team has ML experience' }
            ],
            riskLevel: 'medium'
          }
        },
        {
          id: 'task-2',
          title: 'Design risk overlay UI component',
          description: 'Create visual indicators for risk levels on the Kanban board',
          status: 'todo',
          priority: 'medium',
          assigneeId: 'user-2',
          assigneeName: 'Bob Smith',
          estimatedHours: 8,
          tags: ['Frontend', 'UI', 'React'],
          createdAt: '2025-01-20T10:00:00Z',
          updatedAt: '2025-01-20T10:00:00Z',
          position: 1,
          columnId: 'col-todo',
          boardId: 'board-1',
          riskLevel: 'low'
        }
      ],
      riskLevel: 'medium'
    },
    {
      id: 'col-progress',
      title: 'In Progress',
      status: 'in-progress',
      position: 1,
      boardId: 'board-1',
      wipLimit: 5,
      tasks: [
        {
          id: 'task-3',
          title: 'Implement WebSocket event streaming',
          description: 'Real-time task updates using Socket.IO',
          status: 'in-progress',
          priority: 'high',
          assigneeId: 'user-3',
          assigneeName: 'Charlie Davis',
          estimatedHours: 12,
          actualHours: 8,
          tags: ['Backend', 'WebSocket', 'Real-time'],
          createdAt: '2025-01-19T14:00:00Z',
          updatedAt: '2025-01-21T11:00:00Z',
          position: 0,
          columnId: 'col-progress',
          boardId: 'board-1',
          riskLevel: 'low',
          velocityPrediction: {
            estimatedCompletion: '2025-01-22T16:00:00Z',
            confidence: 0.92,
            factors: [
              { name: 'Progress', impact: 0.5, description: '67% complete' },
              { name: 'Velocity', impact: 0.2, description: 'Ahead of schedule' }
            ],
            riskLevel: 'low'
          }
        }
      ],
      riskLevel: 'low'
    },
    {
      id: 'col-review',
      title: 'Review',
      status: 'review',
      position: 2,
      boardId: 'board-1',
      wipLimit: 3,
      tasks: [
        {
          id: 'task-4',
          title: 'Agent reallocation algorithm',
          description: 'Autonomous task reassignment based on team capacity',
          status: 'review',
          priority: 'urgent',
          assigneeId: 'user-1',
          assigneeName: 'Alice Johnson',
          estimatedHours: 20,
          actualHours: 22,
          tags: ['AI', 'Algorithm', 'Backend'],
          createdAt: '2025-01-18T09:00:00Z',
          updatedAt: '2025-01-21T15:00:00Z',
          position: 0,
          columnId: 'col-review',
          boardId: 'board-1',
          riskLevel: 'high',
          blockers: ['Waiting for code review', 'Performance testing pending']
        }
      ],
      riskLevel: 'high'
    },
    {
      id: 'col-done',
      title: 'Done',
      status: 'done',
      position: 3,
      boardId: 'board-1',
      tasks: [
        {
          id: 'task-5',
          title: 'Setup RabbitMQ event bus',
          description: 'Configure message queue for event sourcing',
          status: 'done',
          priority: 'high',
          assigneeId: 'user-3',
          assigneeName: 'Charlie Davis',
          estimatedHours: 6,
          actualHours: 5,
          tags: ['DevOps', 'Infrastructure', 'RabbitMQ'],
          createdAt: '2025-01-17T10:00:00Z',
          updatedAt: '2025-01-19T16:00:00Z',
          completedAt: '2025-01-19T16:00:00Z',
          position: 0,
          columnId: 'col-done',
          boardId: 'board-1',
          riskLevel: 'low'
        }
      ],
      riskLevel: 'low'
    }
  ],
  teamMembers: [
    {
      id: 'user-1',
      name: 'Alice Johnson',
      email: 'alice@kanopt.dev',
      role: 'developer',
      capacity: 40,
      currentLoad: 36
    },
    {
      id: 'user-2',
      name: 'Bob Smith',
      email: 'bob@kanopt.dev',
      role: 'designer',
      capacity: 35,
      currentLoad: 8
    },
    {
      id: 'user-3',
      name: 'Charlie Davis',
      email: 'charlie@kanopt.dev',
      role: 'developer',
      capacity: 40,
      currentLoad: 17
    }
  ],
  createdAt: '2025-01-15T09:00:00Z',
  updatedAt: '2025-01-21T15:00:00Z'
};

export default function HomePage() {
  const [activeView, setActiveView] = useState<'board' | 'analytics'>('board');
  const [showAgentPanel, setShowAgentPanel] = useState(true);
  
  const {
    setCurrentBoard,
    showRiskOverlay,
    setShowRiskOverlay,
    showVelocityPredictions,
    setShowVelocityPredictions,
    isLoading,
    error
  } = useKanbanStore();

  useEffect(() => {
    setCurrentBoard(mockBoard);
  }, [setCurrentBoard]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="loading-spinner">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          <p className="mt-4 text-gray-600">Loading Kanban board...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="text-red-500 text-xl mb-4">⚠️ Error</div>
          <p className="text-gray-600 mb-4">{error}</p>
          <button 
            className="btn-primary"
            onClick={() => window.location.reload()}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Control Bar */}
      <div className="bg-white border-b border-gray-200 px-6 py-4 relative z-40">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <h2 className="text-xl font-semibold text-gray-900">
              {mockBoard.name}
            </h2>
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setActiveView('board')}
                className={`px-3 py-2 text-sm font-medium rounded-md ${
                  activeView === 'board'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                Board
              </button>
              <button
                onClick={() => setActiveView('analytics')}
                className={`px-3 py-2 text-sm font-medium rounded-md ${
                  activeView === 'analytics'
                    ? 'bg-blue-100 text-blue-700'
                    : 'text-gray-600 hover:text-gray-900'
                }`}
              >
                Analytics
              </button>
            </div>
          </div>

          <div className="flex items-center space-x-4">
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={showRiskOverlay}
                onChange={(e) => setShowRiskOverlay(e.target.checked)}
                className="mr-2"
              />
              <span className="text-sm text-gray-700">Risk Overlay</span>
            </label>
            
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={showVelocityPredictions}
                onChange={(e) => setShowVelocityPredictions(e.target.checked)}
                className="mr-2"
              />
              <span className="text-sm text-gray-700">AI Predictions</span>
            </label>

            <button
              onClick={() => setShowAgentPanel(!showAgentPanel)}
              className={`px-3 py-2 text-sm font-medium rounded-md ${
                showAgentPanel
                  ? 'bg-green-100 text-green-700'
                  : 'bg-gray-100 text-gray-700'
              }`}
            >
              🤖 Agent Panel
            </button>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex overflow-hidden relative">
        {/* Main View */}
        <div className={`flex-1 ${showAgentPanel ? 'pr-80' : ''}`}>
          {activeView === 'board' ? (
            <KanbanBoard boardId={mockBoard.id} />
          ) : (
            <AnalyticsDashboard boardId={mockBoard.id} />
          )}
        </div>

        {/* Agent Panel */}
        {showAgentPanel && (
          <div className="sidebar-panel w-80">
            <AgentPanel boardId={mockBoard.id} />
          </div>
        )}
      </div>
    </div>
  );
}
