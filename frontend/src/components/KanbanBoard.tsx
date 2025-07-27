'use client';

import React, { useEffect, useRef, useCallback } from 'react';
import { DragDropContext, Droppable, Draggable, DropResult } from 'react-beautiful-dnd';
import { motion, AnimatePresence } from 'framer-motion';
import { useKanbanStore } from '@/store/kanban';
import { Task, Column } from '@/types';
import TaskCard from './TaskCard';
import VirtualColumnList from './VirtualColumnList';
import RiskOverlay from './RiskOverlay';
import { useSocket } from '@/hooks/useSocket';
import { useVirtualScroll } from '@/hooks/useVirtualScroll';

interface KanbanBoardProps {
  boardId: string;
  className?: string;
}

const KanbanBoard: React.FC<KanbanBoardProps> = ({ boardId, className }) => {
  const {
    currentBoard,
    dragState,
    showRiskOverlay,
    setDragState,
    clearDragState,
    moveTask,
    addEvent,
  } = useKanbanStore();

  const boardRef = useRef<HTMLDivElement>(null);
  const { socket, emitTaskMoved, emitTaskUpdated, joinBoard, leaveBoard } = useSocket();

  // Virtual scrolling for columns with many tasks
  const {
    containerRef,
    visibleRange,
    scrollToItem,
    isItemVisible,
  } = useVirtualScroll({
    itemHeight: 120,
    itemCount: currentBoard?.columns?.length || 0,
    overscan: 3,
  });

  // Handle drag end
  const handleDragEnd = useCallback((result: DropResult) => {
    const { destination, source, draggableId } = result;

    clearDragState();

    if (!destination) {
      return;
    }

    if (
      destination.droppableId === source.droppableId &&
      destination.index === source.index
    ) {
      return;
    }

    // Move task
    moveTask(
      draggableId,
      source.droppableId,
      destination.droppableId,
      destination.index
    );

    // Emit event
    const event = {
      id: `event-${Date.now()}`,
      type: 'task-moved' as const,
      taskId: draggableId,
      boardId: boardId,
      userId: 'current-user',
      timestamp: new Date().toISOString(),
      data: {
        fromColumnId: source.droppableId,
        toColumnId: destination.droppableId,
        fromIndex: source.index,
        toIndex: destination.index,
      },
    };

    addEvent(event);

    // Send to socket for real-time updates
    emitTaskMoved(event);
  }, [boardId, moveTask, addEvent, clearDragState, emitTaskMoved]);

  // Handle drag start
  const handleDragStart = useCallback((start: any) => {
    const task = currentBoard?.columns
      .flatMap(col => col.tasks)
      .find(task => task.id === start.draggableId);

    if (task) {
      setDragState({
        isDragging: true,
        draggedTask: task,
        sourceColumn: start.source.droppableId,
      });
    }
  }, [currentBoard, setDragState]);

  // Handle drag update for visual feedback
  const handleDragUpdate = useCallback((update: any) => {
    if (update.destination) {
      setDragState({
        targetColumn: update.destination.droppableId,
      });
    }
  }, [setDragState]);

  // Optimize rendering with RAF
  useEffect(() => {
    let rafId: number;
    
    const updateDragPosition = () => {
      if (dragState.isDragging) {
        rafId = requestAnimationFrame(updateDragPosition);
      }
    };

    if (dragState.isDragging) {
      rafId = requestAnimationFrame(updateDragPosition);
    }

    return () => {
      if (rafId) {
        cancelAnimationFrame(rafId);
      }
    };
  }, [dragState.isDragging]);

  if (!currentBoard) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-gray-500">Loading board...</div>
      </div>
    );
  }

  return (
    <div
      ref={boardRef}
      className={`kanban-board relative h-full overflow-hidden ${className || ''}`}
    >
      {/* Risk Overlay */}
      <AnimatePresence>
        {showRiskOverlay && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="absolute inset-0 z-10 pointer-events-none"
          >
            <RiskOverlay columns={currentBoard.columns} />
          </motion.div>
        )}
      </AnimatePresence>

      {/* Main Board */}
      <DragDropContext
        onDragEnd={handleDragEnd}
        onDragStart={handleDragStart}
        onDragUpdate={handleDragUpdate}
      >
        <div
          ref={containerRef}
          className="flex h-full overflow-x-auto overflow-y-hidden gap-6 p-6 scrollbar-hide"
          style={{
            minWidth: `${currentBoard.columns.length * 320}px`,
          }}
        >
          {currentBoard.columns.map((column, index) => (
            <KanbanColumn
              key={column.id}
              column={column}
              index={index}
              isVisible={isItemVisible(index)}
            />
          ))}
        </div>
      </DragDropContext>

      {/* Drag Preview */}
      <AnimatePresence>
        {dragState.isDragging && dragState.draggedTask && (
          <motion.div
            initial={{ scale: 0.95, opacity: 0.8 }}
            animate={{ scale: 1.05, opacity: 0.9 }}
            exit={{ scale: 0.95, opacity: 0 }}
            className="fixed top-0 left-0 z-50 pointer-events-none"
            style={{
              transform: `translate(${dragState.dragPosition?.x || 0}px, ${dragState.dragPosition?.y || 0}px)`,
            }}
          >
            <div className="drag-preview">
              <TaskCard
                task={dragState.draggedTask}
                index={0}
                isDragging={true}
              />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

interface KanbanColumnProps {
  column: Column;
  index: number;
  isVisible: boolean;
}

const KanbanColumn: React.FC<KanbanColumnProps> = ({ column, index, isVisible }) => {
  const { showVelocityPredictions } = useKanbanStore();

  if (!isVisible) {
    return <div className="w-80 flex-shrink-0" />; // Placeholder for non-visible columns
  }

  return (
    <motion.div
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ delay: index * 0.1 }}
      className="column w-80 flex-shrink-0 bg-gray-50 rounded-lg"
    >
      {/* Column Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-gray-900">{column.title}</h3>
            <span className="px-2 py-1 text-xs bg-gray-200 text-gray-600 rounded-full">
              {column.tasks.length}
            </span>
            {column.wipLimit && column.tasks.length > column.wipLimit && (
              <span className="px-2 py-1 text-xs bg-red-100 text-red-600 rounded-full">
                WIP exceeded
              </span>
            )}
          </div>
          
          {showVelocityPredictions && column.riskLevel && (
            <div className={`w-3 h-3 rounded-full risk-${column.riskLevel}`} />
          )}
        </div>
        
        {column.wipLimit && (
          <div className="mt-2">
            <div className="w-full bg-gray-200 rounded-full h-1.5">
              <div
                className={`h-1.5 rounded-full transition-all duration-300 ${
                  column.tasks.length > column.wipLimit
                    ? 'bg-red-500'
                    : column.tasks.length > column.wipLimit * 0.8
                    ? 'bg-yellow-500'
                    : 'bg-green-500'
                }`}
                style={{
                  width: `${Math.min((column.tasks.length / column.wipLimit) * 100, 100)}%`,
                }}
              />
            </div>
          </div>
        )}
      </div>

      {/* Tasks List */}
      <Droppable droppableId={column.id}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={`flex-1 p-4 min-h-[200px] transition-colors duration-200 ${
              snapshot.isDraggingOver ? 'drop-zone' : ''
            }`}
          >
            <VirtualColumnList
              tasks={column.tasks}
              columnId={column.id}
              isDraggingOver={snapshot.isDraggingOver}
            />
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </motion.div>
  );
};

export default KanbanBoard;
