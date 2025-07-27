'use client';

import React from 'react';
import { Draggable } from 'react-beautiful-dnd';
import { motion } from 'framer-motion';
import { Clock, User, AlertTriangle, TrendingUp } from 'lucide-react';
import { Task } from '@/types';
import { useKanbanStore } from '@/store/kanban';

interface TaskCardProps {
  task: Task;
  index: number;
  isDragging?: boolean;
}

const TaskCard: React.FC<TaskCardProps> = ({ task, index, isDragging = false }) => {
  const { showVelocityPredictions, setSelectedTask } = useKanbanStore();

  const getRiskColor = (riskLevel?: string) => {
    switch (riskLevel) {
      case 'low': return 'border-green-400 bg-green-50';
      case 'medium': return 'border-yellow-400 bg-yellow-50';
      case 'high': return 'border-orange-400 bg-orange-50';
      case 'critical': return 'border-red-400 bg-red-50';
      default: return 'border-gray-200 bg-white';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent': return 'bg-red-500';
      case 'high': return 'bg-orange-500';
      case 'medium': return 'bg-yellow-500';
      case 'low': return 'bg-green-500';
      default: return 'bg-gray-500';
    }
  };

  const formatEstimate = (hours?: number) => {
    if (!hours) return null;
    if (hours < 1) return `${hours * 60}m`;
    return `${hours}h`;
  };

  const handleClick = () => {
    setSelectedTask(task.id);
  };

  const cardContent = (
    <motion.div
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      className={`
        task-card p-4 rounded-lg border-2 cursor-pointer transition-all duration-200
        ${getRiskColor(task.riskLevel)}
        ${isDragging ? 'shadow-lg rotate-2 scale-105' : 'shadow-sm hover:shadow-md'}
      `}
      onClick={handleClick}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <div 
            className={`w-3 h-3 rounded-full ${getPriorityColor(task.priority)}`}
            title={`Priority: ${task.priority}`}
          />
          {task.riskLevel && showVelocityPredictions && (
            <div title={`Risk: ${task.riskLevel}`}>
              <AlertTriangle 
                className={`w-4 h-4 ${
                  task.riskLevel === 'critical' ? 'text-red-500' :
                  task.riskLevel === 'high' ? 'text-orange-500' :
                  task.riskLevel === 'medium' ? 'text-yellow-500' :
                  'text-green-500'
                }`}
              />
            </div>
          )}
        </div>
        
        {task.velocityPrediction && showVelocityPredictions && (
          <div className="flex items-center gap-1 text-xs text-gray-600">
            <TrendingUp className="w-3 h-3" />
            {Math.round(task.velocityPrediction.confidence * 100)}%
          </div>
        )}
      </div>

      {/* Title */}
      <h4 className="font-medium text-gray-900 mb-2 line-clamp-2">
        {task.title}
      </h4>

      {/* Description */}
      {task.description && (
        <p className="text-sm text-gray-600 mb-3 line-clamp-2">
          {task.description}
        </p>
      )}

      {/* Tags */}
      {task.tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mb-3">
          {task.tags.slice(0, 3).map((tag, index) => (
            <span
              key={index}
              className="px-2 py-1 text-xs bg-blue-100 text-blue-700 rounded-full"
            >
              {tag}
            </span>
          ))}
          {task.tags.length > 3 && (
            <span className="px-2 py-1 text-xs bg-gray-100 text-gray-600 rounded-full">
              +{task.tags.length - 3}
            </span>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between text-xs text-gray-500">
        <div className="flex items-center gap-3">
          {/* Estimate */}
          {task.estimatedHours && (
            <div className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {formatEstimate(task.estimatedHours)}
            </div>
          )}
          
          {/* Assignee */}
          {task.assigneeName && (
            <div className="flex items-center gap-1">
              <User className="w-3 h-3" />
              <span className="truncate max-w-[60px]">
                {task.assigneeName}
              </span>
            </div>
          )}
        </div>

        {/* Velocity Prediction */}
        {task.velocityPrediction && showVelocityPredictions && (
          <div className="text-xs">
            <span className={`
              px-2 py-1 rounded-full
              ${task.velocityPrediction.confidence > 0.8 ? 'bg-green-100 text-green-700' :
                task.velocityPrediction.confidence > 0.6 ? 'bg-yellow-100 text-yellow-700' :
                'bg-red-100 text-red-700'}
            `}>
              {new Date(task.velocityPrediction.estimatedCompletion).toLocaleDateString()}
            </span>
          </div>
        )}
      </div>

      {/* Progress indicator for in-progress tasks */}
      {task.status === 'in-progress' && task.actualHours && task.estimatedHours && (
        <div className="mt-3">
          <div className="flex justify-between text-xs text-gray-600 mb-1">
            <span>Progress</span>
            <span>{Math.round((task.actualHours / task.estimatedHours) * 100)}%</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-1.5">
            <div
              className={`h-1.5 rounded-full transition-all duration-300 ${
                task.actualHours > task.estimatedHours
                  ? 'bg-red-500'
                  : 'bg-blue-500'
              }`}
              style={{
                width: `${Math.min((task.actualHours / task.estimatedHours) * 100, 100)}%`,
              }}
            />
          </div>
        </div>
      )}

      {/* Blockers indicator */}
      {task.blockers && task.blockers.length > 0 && (
        <div className="mt-2 text-xs text-red-600 bg-red-50 px-2 py-1 rounded">
          🚫 {task.blockers.length} blocker{task.blockers.length > 1 ? 's' : ''}
        </div>
      )}
    </motion.div>
  );

  if (isDragging) {
    return cardContent;
  }

  return (
    <Draggable draggableId={task.id} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          className={`mb-3 ${snapshot.isDragging ? 'z-50' : ''}`}
        >
          {cardContent}
        </div>
      )}
    </Draggable>
  );
};

export default TaskCard;
