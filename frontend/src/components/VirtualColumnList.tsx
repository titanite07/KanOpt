'use client';

import React, { useMemo } from 'react';
import { FixedSizeList as List } from 'react-window';
import { Task } from '@/types';
import TaskCard from './TaskCard';

interface VirtualColumnListProps {
  tasks: Task[];
  columnId: string;
  isDraggingOver: boolean;
}

const VirtualColumnList: React.FC<VirtualColumnListProps> = ({ 
  tasks, 
  columnId, 
  isDraggingOver 
}) => {
  const ITEM_HEIGHT = 120;
  const MAX_HEIGHT = 600;

  const sortedTasks = useMemo(() => {
    return [...tasks].sort((a, b) => a.position - b.position);
  }, [tasks]);

  const listHeight = useMemo(() => {
    return Math.min(sortedTasks.length * ITEM_HEIGHT, MAX_HEIGHT);
  }, [sortedTasks.length]);

  const renderTaskItem = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const task = sortedTasks[index];
    
    return (
      <div style={style}>
        <TaskCard 
          task={task} 
          index={index}
        />
      </div>
    );
  };

  if (sortedTasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-32 text-gray-400 text-sm">
        {isDraggingOver ? 'Drop task here' : 'No tasks'}
      </div>
    );
  }

  // For small lists, render normally to avoid virtualization overhead
  if (sortedTasks.length <= 5) {
    return (
      <div className="space-y-3">
        {sortedTasks.map((task, index) => (
          <TaskCard 
            key={task.id}
            task={task} 
            index={index}
          />
        ))}
      </div>
    );
  }

  return (
    <List
      height={listHeight}
      width="100%"
      itemCount={sortedTasks.length}
      itemSize={ITEM_HEIGHT}
      itemData={sortedTasks}
      className="scrollbar-hide"
    >
      {renderTaskItem}
    </List>
  );
};

export default VirtualColumnList;
