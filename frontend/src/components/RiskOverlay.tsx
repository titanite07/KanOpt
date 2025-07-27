'use client';

import React from 'react';
import { motion } from 'framer-motion';
import { Column } from '@/types';

interface RiskOverlayProps {
  columns: Column[];
}

const RiskOverlay: React.FC<RiskOverlayProps> = ({ columns }) => {
  const getRiskIntensity = (riskLevel?: string) => {
    switch (riskLevel) {
      case 'critical': return 0.8;
      case 'high': return 0.6;
      case 'medium': return 0.4;
      case 'low': return 0.2;
      default: return 0;
    }
  };

  const getRiskColor = (riskLevel?: string) => {
    switch (riskLevel) {
      case 'critical': return 'rgba(239, 68, 68, ';
      case 'high': return 'rgba(245, 158, 11, ';
      case 'medium': return 'rgba(234, 179, 8, ';
      case 'low': return 'rgba(34, 197, 94, ';
      default: return 'rgba(107, 114, 128, ';
    }
  };

  return (
    <div className="absolute inset-0 pointer-events-none">
      <div className="flex h-full gap-6 p-6">
        {columns.map((column, index) => {
          const intensity = getRiskIntensity(column.riskLevel);
          const color = getRiskColor(column.riskLevel);
          
          return (
            <motion.div
              key={column.id}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: index * 0.1 }}
              className="w-80 flex-shrink-0 rounded-lg relative"
              style={{
                backgroundColor: `${color}${intensity})`,
                border: intensity > 0.5 ? `2px solid ${color}0.8)` : 'none',
              }}
            >
              {/* Risk Level Indicator */}
              {column.riskLevel && (
                <motion.div
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ delay: index * 0.1 + 0.2 }}
                  className="absolute top-2 right-2 px-2 py-1 rounded-full text-xs font-medium text-white"
                  style={{
                    backgroundColor: `${color}0.9)`,
                  }}
                >
                  {column.riskLevel.toUpperCase()}
                </motion.div>
              )}

              {/* Pulsing animation for high risk */}
              {(column.riskLevel === 'high' || column.riskLevel === 'critical') && (
                <motion.div
                  className="absolute inset-0 rounded-lg"
                  style={{
                    backgroundColor: `${color}0.2)`,
                  }}
                  animate={{
                    opacity: [0, 0.5, 0],
                  }}
                  transition={{
                    duration: 2,
                    repeat: Infinity,
                    ease: "easeInOut",
                  }}
                />
              )}

              {/* Risk metrics overlay for tasks */}
              <div className="absolute bottom-2 left-2 right-2">
                {column.tasks.map((task, taskIndex) => {
                  if (!task.riskLevel) return null;
                  
                  const taskIntensity = getRiskIntensity(task.riskLevel);
                  
                  return (
                    <motion.div
                      key={task.id}
                      initial={{ scale: 0, opacity: 0 }}
                      animate={{ scale: 1, opacity: 1 }}
                      transition={{ delay: index * 0.1 + taskIndex * 0.05 }}
                      className="mb-1 h-2 rounded-full"
                      style={{
                        backgroundColor: `${getRiskColor(task.riskLevel)}${taskIntensity})`,
                        width: `${Math.min(taskIntensity * 100 + 20, 100)}%`,
                      }}
                    />
                  );
                })}
              </div>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
};

export default RiskOverlay;
