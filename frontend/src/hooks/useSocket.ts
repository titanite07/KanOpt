import { useEffect, useRef, useState } from 'react';
import { io, Socket } from 'socket.io-client';
import { useKanbanStore } from '@/store/kanban';
import { TaskEvent, RiskAlert, ReallocationSuggestion } from '@/types';

interface UseSocketOptions {
  autoConnect?: boolean;
  reconnection?: boolean;
  timeout?: number;
}

export const useSocket = (options: UseSocketOptions = {}) => {
  const [socket, setSocket] = useState<Socket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const {
    addEvent,
    addRiskAlert,
    addSuggestion,
    updateTask,
    setError: setStoreError,
  } = useKanbanStore();

  const socketRef = useRef<Socket | null>(null);

  useEffect(() => {
    const {
      autoConnect = true,
      reconnection = true,
      timeout = 20000,
    } = options;

    if (!autoConnect) return;

    const socketInstance = io(process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080', {
      reconnection,
      timeout,
      transports: ['websocket', 'polling'],
    });

    socketRef.current = socketInstance;
    setSocket(socketInstance);

    // Connection events
    socketInstance.on('connect', () => {
      setIsConnected(true);
      setError(null);
      console.log('Socket connected:', socketInstance.id);
    });

    socketInstance.on('disconnect', (reason) => {
      setIsConnected(false);
      console.log('Socket disconnected:', reason);
    });

    socketInstance.on('connect_error', (error) => {
      setError(error.message);
      setStoreError(`WebSocket connection failed: ${error.message}`);
      console.error('Socket connection error:', error);
    });

    // Kanban events
    socketInstance.on('task-moved', (event: TaskEvent) => {
      addEvent(event);
    });

    socketInstance.on('task-updated', (event: TaskEvent) => {
      addEvent(event);
      if (event.newState) {
        updateTask(event.taskId, event.newState);
      }
    });

    socketInstance.on('risk-alert', (alert: RiskAlert) => {
      addRiskAlert(alert);
    });

    socketInstance.on('reallocation-suggested', (suggestion: ReallocationSuggestion) => {
      addSuggestion(suggestion);
    });

    // Board sync events
    socketInstance.on('board-updated', (boardData) => {
      console.log('Board updated:', boardData);
      // Handle board state synchronization
    });

    socketInstance.on('user-joined', (userData) => {
      console.log('User joined:', userData);
    });

    socketInstance.on('user-left', (userData) => {
      console.log('User left:', userData);
    });

    return () => {
      socketInstance.disconnect();
      socketRef.current = null;
    };
  }, [options, addEvent, addRiskAlert, addSuggestion, updateTask, setStoreError]);

  // Emit events
  const emitTaskMoved = (event: TaskEvent) => {
    if (socket?.connected) {
      socket.emit('task-moved', event);
    }
  };

  const emitTaskUpdated = (event: TaskEvent) => {
    if (socket?.connected) {
      socket.emit('task-updated', event);
    }
  };

  const joinBoard = (boardId: string) => {
    if (socket?.connected) {
      socket.emit('join-board', boardId);
    }
  };

  const leaveBoard = (boardId: string) => {
    if (socket?.connected) {
      socket.emit('leave-board', boardId);
    }
  };

  return {
    socket,
    isConnected,
    error,
    emitTaskMoved,
    emitTaskUpdated,
    joinBoard,
    leaveBoard,
  };
};
