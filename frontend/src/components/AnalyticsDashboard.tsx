'use client';

import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, LineChart, Line, PieChart, Pie, Cell } from 'recharts';
import { TrendingUp, TrendingDown, Activity, Users, Clock, Target } from 'lucide-react';
import { useKanbanStore } from '@/store/kanban';

interface AnalyticsDashboardProps {
  boardId: string;
}

const velocityData = [
  { week: 'W1', planned: 20, actual: 18, prediction: 19 },
  { week: 'W2', planned: 25, actual: 22, prediction: 24 },
  { week: 'W3', planned: 22, actual: 26, prediction: 23 },
  { week: 'W4', planned: 28, actual: 24, prediction: 27 },
  { week: 'W5', planned: 30, actual: 0, prediction: 29 },
];

const burndownData = [
  { day: 'Day 1', remaining: 100, ideal: 95 },
  { day: 'Day 2', remaining: 88, ideal: 90 },
  { day: 'Day 3', remaining: 82, ideal: 85 },
  { day: 'Day 4', remaining: 75, ideal: 80 },
  { day: 'Day 5', remaining: 68, ideal: 75 },
  { day: 'Day 6', remaining: 60, ideal: 70 },
  { day: 'Day 7', remaining: 52, ideal: 65 },
];

const riskDistribution = [
  { name: 'Low', value: 45, color: '#10b981' },
  { name: 'Medium', value: 30, color: '#f59e0b' },
  { name: 'High', value: 20, color: '#ef4444' },
  { name: 'Critical', value: 5, color: '#dc2626' },
];

const teamPerformance = [
  { name: 'Alice', completed: 12, capacity: 15, efficiency: 80 },
  { name: 'Bob', completed: 8, capacity: 10, efficiency: 80 },
  { name: 'Charlie', completed: 14, capacity: 15, efficiency: 93 },
];

const AnalyticsDashboard: React.FC<AnalyticsDashboardProps> = ({ boardId }) => {
  const [timeRange, setTimeRange] = useState<'week' | 'month' | 'quarter'>('week');
  const { currentBoard } = useKanbanStore();

  if (!currentBoard) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-gray-500">No board data available</div>
      </div>
    );
  }

  const totalTasks = currentBoard.columns.reduce((acc, col) => acc + col.tasks.length, 0);
  const completedTasks = currentBoard.columns.find(col => col.status === 'done')?.tasks.length || 0;
  const inProgressTasks = currentBoard.columns.find(col => col.status === 'in-progress')?.tasks.length || 0;
  const blockedTasks = currentBoard.columns.reduce((acc, col) => 
    acc + col.tasks.filter(task => task.blockers && task.blockers.length > 0).length, 0
  );

  return (
    <div className="h-full overflow-y-auto bg-gray-50 p-6">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-gray-900">Analytics Dashboard</h2>
          <div className="flex items-center space-x-4">
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value as any)}
              className="px-3 py-2 border border-gray-300 rounded-md text-sm"
            >
              <option value="week">This Week</option>
              <option value="month">This Month</option>
              <option value="quarter">This Quarter</option>
            </select>
          </div>
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <KPICard
          title="Total Tasks"
          value={totalTasks}
          change={12}
          icon={<Activity className="w-6 h-6" />}
          color="blue"
        />
        <KPICard
          title="Completed"
          value={completedTasks}
          change={8}
          icon={<Target className="w-6 h-6" />}
          color="green"
        />
        <KPICard
          title="In Progress"
          value={inProgressTasks}
          change={-2}
          icon={<Clock className="w-6 h-6" />}
          color="yellow"
        />
        <KPICard
          title="Blocked"
          value={blockedTasks}
          change={-1}
          icon={<Users className="w-6 h-6" />}
          color="red"
        />
      </div>

      {/* Charts Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Velocity Chart */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Velocity Trends</h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={velocityData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="week" />
              <YAxis />
              <Tooltip />
              <Bar dataKey="planned" fill="#e5e7eb" name="Planned" />
              <Bar dataKey="actual" fill="#3b82f6" name="Actual" />
              <Bar dataKey="prediction" fill="#10b981" name="AI Prediction" />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Burndown Chart */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Sprint Burndown</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={burndownData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="day" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="ideal" stroke="#e5e7eb" strokeDasharray="5 5" name="Ideal" />
              <Line type="monotone" dataKey="remaining" stroke="#3b82f6" strokeWidth={2} name="Actual" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Risk and Team Performance */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Risk Distribution */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Risk Distribution</h3>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={riskDistribution}
                cx="50%"
                cy="50%"
                outerRadius={80}
                dataKey="value"
                label={(props: any) => `${props.name}: ${((props.percent || 0) * 100).toFixed(0)}%`}
              >
                {riskDistribution.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>

        {/* Team Performance */}
        <div className="bg-white rounded-lg border border-gray-200 p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Team Performance</h3>
          <div className="space-y-4">
            {teamPerformance.map((member, index) => (
              <div key={index} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-gray-300 rounded-full flex items-center justify-center text-sm font-medium text-white">
                    {member.name[0]}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900">{member.name}</p>
                    <p className="text-sm text-gray-500">
                      {member.completed}/{member.capacity} tasks
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <div className={`text-sm font-medium ${
                    member.efficiency >= 90 ? 'text-green-600' :
                    member.efficiency >= 80 ? 'text-yellow-600' :
                    'text-red-600'
                  }`}>
                    {member.efficiency}%
                  </div>
                  <div className="w-20 bg-gray-200 rounded-full h-2 mt-1">
                    <div
                      className={`h-2 rounded-full ${
                        member.efficiency >= 90 ? 'bg-green-500' :
                        member.efficiency >= 80 ? 'bg-yellow-500' :
                        'bg-red-500'
                      }`}
                      style={{ width: `${member.efficiency}%` }}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* AI Insights Panel */}
      <div className="mt-8 bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg border border-blue-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
          <Activity className="w-5 h-5 mr-2 text-blue-600" />
          AI Insights
        </h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
          <div className="bg-white rounded-lg p-4">
            <h4 className="font-medium text-gray-900 mb-2">Velocity Prediction</h4>
            <p className="text-gray-600">
              Based on current trends, the team is likely to complete{' '}
              <span className="font-medium text-blue-600">24-26 story points</span> this sprint.
            </p>
          </div>
          <div className="bg-white rounded-lg p-4">
            <h4 className="font-medium text-gray-900 mb-2">Risk Assessment</h4>
            <p className="text-gray-600">
              High complexity tasks are at{' '}
              <span className="font-medium text-orange-600">medium risk</span> of delay.
              Consider reassigning resources.
            </p>
          </div>
          <div className="bg-white rounded-lg p-4">
            <h4 className="font-medium text-gray-900 mb-2">Team Optimization</h4>
            <p className="text-gray-600">
              Charlie is operating at{' '}
              <span className="font-medium text-green-600">93% efficiency</span>.
              Consider balancing workload with other team members.
            </p>
          </div>
          <div className="bg-white rounded-lg p-4">
            <h4 className="font-medium text-gray-900 mb-2">Bottleneck Detection</h4>
            <p className="text-gray-600">
              Code review process is creating a{' '}
              <span className="font-medium text-red-600">bottleneck</span>.
              Consider parallel review streams.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

// KPI Card Component
const KPICard = ({ title, value, change, icon, color }: {
  title: string;
  value: number;
  change: number;
  icon: React.ReactNode;
  color: 'blue' | 'green' | 'yellow' | 'red';
}) => {
  const colorClasses = {
    blue: 'bg-blue-500 text-blue-600',
    green: 'bg-green-500 text-green-600',
    yellow: 'bg-yellow-500 text-yellow-600',
    red: 'bg-red-500 text-red-600',
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white rounded-lg border border-gray-200 p-6"
    >
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600">{title}</p>
          <p className="text-2xl font-bold text-gray-900">{value}</p>
        </div>
        <div className={`w-12 h-12 rounded-lg bg-opacity-10 flex items-center justify-center ${colorClasses[color]}`}>
          {icon}
        </div>
      </div>
      <div className="mt-4 flex items-center">
        {change >= 0 ? (
          <TrendingUp className="w-4 h-4 text-green-500 mr-1" />
        ) : (
          <TrendingDown className="w-4 h-4 text-red-500 mr-1" />
        )}
        <span className={`text-sm font-medium ${change >= 0 ? 'text-green-600' : 'text-red-600'}`}>
          {change >= 0 ? '+' : ''}{change}%
        </span>
        <span className="text-sm text-gray-500 ml-1">vs last period</span>
      </div>
    </motion.div>
  );
};

export default AnalyticsDashboard;
