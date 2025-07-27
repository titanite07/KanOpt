'use client';

import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Bot, Brain, AlertTriangle, CheckCircle, XCircle, Clock, TrendingUp } from 'lucide-react';
import { useKanbanStore } from '@/store/kanban';
import { ReallocationSuggestion, AgentAction, RiskAlert } from '@/types';

interface AgentPanelProps {
  boardId: string;
}

const AgentPanel: React.FC<AgentPanelProps> = ({ boardId }) => {
  const [activeTab, setActiveTab] = useState<'suggestions' | 'actions' | 'alerts'>('suggestions');
  const [agentStatus, setAgentStatus] = useState<'active' | 'idle' | 'learning'>('active');
  
  const {
    suggestions,
    agentActions,
    riskAlerts,
    approveSuggestion,
    rejectSuggestion,
    approveAgentAction,
    rejectAgentAction,
    acknowledgeRiskAlert,
    getPendingSuggestions,
    getPendingAgentActions,
    getRiskAlertsByLevel,
  } = useKanbanStore();

  const pendingSuggestions = getPendingSuggestions();
  const pendingActions = getPendingAgentActions();
  const criticalAlerts = getRiskAlertsByLevel('critical');
  const highAlerts = getRiskAlertsByLevel('high');

  useEffect(() => {
    // Simulate agent activity
    const interval = setInterval(() => {
      const statuses = ['active', 'idle', 'learning'] as const;
      setAgentStatus(statuses[Math.floor(Math.random() * statuses.length)]);
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const getStatusIcon = () => {
    switch (agentStatus) {
      case 'active': return <Bot className="w-5 h-5 text-green-500" />;
      case 'learning': return <Brain className="w-5 h-5 text-blue-500" />;
      default: return <Clock className="w-5 h-5 text-gray-400" />;
    }
  };

  const getStatusText = () => {
    switch (agentStatus) {
      case 'active': return 'Monitoring workflow';
      case 'learning': return 'Training model';
      default: return 'Idle';
    }
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="p-4 bg-white border-b border-gray-200">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-gray-900">AI Agent</h3>
          <div className="flex items-center space-x-2">
            {getStatusIcon()}
            <span className="text-sm text-gray-600">{getStatusText()}</span>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
          {(['suggestions', 'actions', 'alerts'] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`flex-1 px-3 py-2 text-xs font-medium rounded-md transition-colors ${
                activeTab === tab
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
              {tab === 'suggestions' && pendingSuggestions.length > 0 && (
                <span className="ml-1 px-1.5 py-0.5 bg-blue-100 text-blue-600 rounded-full text-xs">
                  {pendingSuggestions.length}
                </span>
              )}
              {tab === 'actions' && pendingActions.length > 0 && (
                <span className="ml-1 px-1.5 py-0.5 bg-green-100 text-green-600 rounded-full text-xs">
                  {pendingActions.length}
                </span>
              )}
              {tab === 'alerts' && (criticalAlerts.length + highAlerts.length) > 0 && (
                <span className="ml-1 px-1.5 py-0.5 bg-red-100 text-red-600 rounded-full text-xs">
                  {criticalAlerts.length + highAlerts.length}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        <AnimatePresence mode="wait">
          {activeTab === 'suggestions' && (
            <SuggestionsTab 
              suggestions={pendingSuggestions}
              onApprove={approveSuggestion}
              onReject={rejectSuggestion}
            />
          )}
          {activeTab === 'actions' && (
            <ActionsTab 
              actions={pendingActions}
              onApprove={approveAgentAction}
              onReject={rejectAgentAction}
            />
          )}
          {activeTab === 'alerts' && (
            <AlertsTab 
              alerts={[...criticalAlerts, ...highAlerts]}
              onAcknowledge={acknowledgeRiskAlert}
            />
          )}
        </AnimatePresence>
      </div>

      {/* Agent Stats */}
      <div className="p-4 bg-white border-t border-gray-200">
        <div className="text-xs text-gray-500 space-y-1">
          <div className="flex justify-between">
            <span>Predictions made:</span>
            <span>247</span>
          </div>
          <div className="flex justify-between">
            <span>Accuracy:</span>
            <span>89.3%</span>
          </div>
          <div className="flex justify-between">
            <span>Actions suggested:</span>
            <span>42</span>
          </div>
          <div className="flex justify-between">
            <span>Accepted:</span>
            <span>38 (90.5%)</span>
          </div>
        </div>
      </div>
    </div>
  );
};

// Suggestions Tab Component
const SuggestionsTab = ({ suggestions, onApprove, onReject }: {
  suggestions: ReallocationSuggestion[];
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
}) => (
  <motion.div
    initial={{ opacity: 0, y: 20 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: -20 }}
    className="p-4 space-y-4"
  >
    {suggestions.length === 0 ? (
      <div className="text-center text-gray-500 py-8">
        <Bot className="w-12 h-12 mx-auto mb-3 text-gray-400" />
        <p>No pending suggestions</p>
        <p className="text-xs">The agent is monitoring for optimization opportunities</p>
      </div>
    ) : (
      suggestions.map((suggestion) => (
        <motion.div
          key={suggestion.id}
          initial={{ scale: 0.95, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          className="bg-white rounded-lg border border-gray-200 p-4"
        >
          <div className="flex items-start justify-between mb-3">
            <h4 className="font-medium text-gray-900 text-sm">Task Reallocation</h4>
            <div className="flex items-center space-x-1">
              <TrendingUp className="w-4 h-4 text-green-500" />
              <span className="text-xs text-green-600">
                {Math.round(suggestion.confidence * 100)}% confidence
              </span>
            </div>
          </div>
          
          <p className="text-sm text-gray-600 mb-3">{suggestion.reasoning}</p>
          
          <div className="text-xs text-gray-500 mb-3 space-y-1">
            <div>Velocity improvement: +{suggestion.impact.velocityImprovement}%</div>
            <div>Risk reduction: -{suggestion.impact.riskReduction}%</div>
            <div>Load balance: +{suggestion.impact.loadBalance}%</div>
          </div>
          
          <div className="flex space-x-2">
            <button
              onClick={() => onApprove(suggestion.id)}
              className="flex-1 bg-green-600 text-white text-xs py-2 px-3 rounded-md hover:bg-green-700 transition-colors"
            >
              <CheckCircle className="w-3 h-3 inline mr-1" />
              Approve
            </button>
            <button
              onClick={() => onReject(suggestion.id)}
              className="flex-1 bg-gray-600 text-white text-xs py-2 px-3 rounded-md hover:bg-gray-700 transition-colors"
            >
              <XCircle className="w-3 h-3 inline mr-1" />
              Reject
            </button>
          </div>
        </motion.div>
      ))
    )}
  </motion.div>
);

// Actions Tab Component
const ActionsTab = ({ actions, onApprove, onReject }: {
  actions: AgentAction[];
  onApprove: (id: string) => void;
  onReject: (id: string) => void;
}) => (
  <motion.div
    initial={{ opacity: 0, y: 20 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: -20 }}
    className="p-4 space-y-4"
  >
    {actions.length === 0 ? (
      <div className="text-center text-gray-500 py-8">
        <Brain className="w-12 h-12 mx-auto mb-3 text-gray-400" />
        <p>No pending actions</p>
        <p className="text-xs">Agent actions will appear here</p>
      </div>
    ) : (
      actions.map((action) => (
        <motion.div
          key={action.id}
          initial={{ scale: 0.95, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          className="bg-white rounded-lg border border-gray-200 p-4"
        >
          <div className="flex items-start justify-between mb-3">
            <h4 className="font-medium text-gray-900 text-sm capitalize">
              {action.type.replace('-', ' ')}
            </h4>
            <span className="text-xs text-blue-600">
              {Math.round(action.confidence * 100)}% confidence
            </span>
          </div>
          
          <p className="text-sm text-gray-600 mb-3">{action.description}</p>
          
          <div className="text-xs text-gray-500 mb-3">
            <div>Expected benefit: {action.impact.estimatedBenefit}</div>
          </div>
          
          <div className="flex space-x-2">
            <button
              onClick={() => onApprove(action.id)}
              className="flex-1 bg-blue-600 text-white text-xs py-2 px-3 rounded-md hover:bg-blue-700 transition-colors"
            >
              Execute
            </button>
            <button
              onClick={() => onReject(action.id)}
              className="flex-1 bg-gray-600 text-white text-xs py-2 px-3 rounded-md hover:bg-gray-700 transition-colors"
            >
              Cancel
            </button>
          </div>
        </motion.div>
      ))
    )}
  </motion.div>
);

// Alerts Tab Component
const AlertsTab = ({ alerts, onAcknowledge }: {
  alerts: RiskAlert[];
  onAcknowledge: (id: string) => void;
}) => (
  <motion.div
    initial={{ opacity: 0, y: 20 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: -20 }}
    className="p-4 space-y-4"
  >
    {alerts.length === 0 ? (
      <div className="text-center text-gray-500 py-8">
        <CheckCircle className="w-12 h-12 mx-auto mb-3 text-green-400" />
        <p>No active alerts</p>
        <p className="text-xs">All systems running smoothly</p>
      </div>
    ) : (
      alerts.map((alert) => (
        <motion.div
          key={alert.id}
          initial={{ scale: 0.95, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          className={`bg-white rounded-lg border-l-4 p-4 ${
            alert.riskLevel === 'critical' ? 'border-red-500' : 'border-orange-500'
          }`}
        >
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center">
              <AlertTriangle className={`w-4 h-4 mr-2 ${
                alert.riskLevel === 'critical' ? 'text-red-500' : 'text-orange-500'
              }`} />
              <span className="font-medium text-gray-900 text-sm capitalize">
                {alert.riskLevel} Risk
              </span>
            </div>
            <span className="text-xs text-gray-500">
              {new Date(alert.timestamp).toLocaleTimeString()}
            </span>
          </div>
          
          <p className="text-sm text-gray-600 mb-3">{alert.message}</p>
          
          {alert.suggestedActions.length > 0 && (
            <div className="mb-3">
              <p className="text-xs text-gray-500 mb-1">Suggested actions:</p>
              <ul className="text-xs text-gray-600 space-y-1">
                {alert.suggestedActions.slice(0, 2).map((suggestion, index) => (
                  <li key={index} className="flex items-start">
                    <span className="w-1 h-1 bg-gray-400 rounded-full mt-2 mr-2 flex-shrink-0" />
                    {suggestion.reasoning}
                  </li>
                ))}
              </ul>
            </div>
          )}
          
          <button
            onClick={() => onAcknowledge(alert.id)}
            className="w-full bg-gray-100 text-gray-700 text-xs py-2 px-3 rounded-md hover:bg-gray-200 transition-colors"
          >
            Acknowledge
          </button>
        </motion.div>
      ))
    )}
  </motion.div>
);

export default AgentPanel;
