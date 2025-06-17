import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Activity, 
  Plus, 
  Settings, 
  Play, 
  Pause, 
  Trash2, 
  Eye,
  AlertTriangle,
  CheckCircle,
  Clock,
  Users,
  Download,
  Upload,
  Cpu,
  HardDrive
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';
import { toast } from 'react-hot-toast';

import { tunnelApi } from '../../api/tunnelApi';
import { TunnelCard } from './TunnelCard';
import { CreateTunnelModal } from './CreateTunnelModal';
import { TunnelDetailsModal } from './TunnelDetailsModal';
import { PerformanceChart } from './PerformanceChart';
import { StatusBadge } from '../UI/StatusBadge';
import { LoadingSpinner } from '../UI/LoadingSpinner';

interface Tunnel {
  id: string;
  name: string;
  description: string;
  protocol: string;
  status: 'active' | 'inactive' | 'error' | 'connecting';
  server_ip: string;
  server_port: number;
  target_ip: string;
  target_port: number;
  is_online: boolean;
  last_ping?: string;
  uptime: string;
  bytes_in: number;
  bytes_out: number;
  connection_count: number;
  performance?: {
    avg_latency: number;
    total_bytes: number;
    bytes_per_sec: number;
    connections_per_sec: number;
    error_rate: number;
  };
  created_at: string;
  updated_at: string;
}

interface DashboardStats {
  total_tunnels: number;
  active_tunnels: number;
  total_bandwidth: number;
  total_connections: number;
  avg_latency: number;
  uptime_percentage: number;
}

export const TunnelDashboard: React.FC = () => {
  const [selectedTunnel, setSelectedTunnel] = useState<Tunnel | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showDetailsModal, setShowDetailsModal] = useState(false);
  const [filter, setFilter] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState('');

  const queryClient = useQueryClient();

  // Fetch tunnels
  const { data: tunnelsData, isLoading: tunnelsLoading, error: tunnelsError } = useQuery({
    queryKey: ['tunnels', filter, searchTerm],
    queryFn: () => tunnelApi.getTunnels({ 
      status: filter !== 'all' ? filter : undefined,
      search: searchTerm || undefined 
    }),
    refetchInterval: 5000, // Refresh every 5 seconds
  });

  // Fetch dashboard stats
  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: () => tunnelApi.getDashboardStats(),
    refetchInterval: 10000, // Refresh every 10 seconds
  });

  // Mutations
  const startTunnelMutation = useMutation({
    mutationFn: (tunnelId: string) => tunnelApi.startTunnel(tunnelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tunnels'] });
      toast.success('Tunnel started successfully');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to start tunnel');
    },
  });

  const stopTunnelMutation = useMutation({
    mutationFn: (tunnelId: string) => tunnelApi.stopTunnel(tunnelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tunnels'] });
      toast.success('Tunnel stopped successfully');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to stop tunnel');
    },
  });

  const deleteTunnelMutation = useMutation({
    mutationFn: (tunnelId: string) => tunnelApi.deleteTunnel(tunnelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tunnels'] });
      toast.success('Tunnel deleted successfully');
    },
    onError: (error: any) => {
      toast.error(error.message || 'Failed to delete tunnel');
    },
  });

  const handleTunnelAction = (action: string, tunnel: Tunnel) => {
    switch (action) {
      case 'start':
        startTunnelMutation.mutate(tunnel.id);
        break;
      case 'stop':
        stopTunnelMutation.mutate(tunnel.id);
        break;
      case 'delete':
        if (window.confirm('Are you sure you want to delete this tunnel?')) {
          deleteTunnelMutation.mutate(tunnel.id);
        }
        break;
      case 'view':
        setSelectedTunnel(tunnel);
        setShowDetailsModal(true);
        break;
      case 'edit':
        setSelectedTunnel(tunnel);
        setShowCreateModal(true);
        break;
    }
  };

  const filteredTunnels = tunnelsData?.data?.filter((tunnel: Tunnel) => {
    const matchesFilter = filter === 'all' || tunnel.status === filter;
    const matchesSearch = tunnel.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         tunnel.description.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesFilter && matchesSearch;
  }) || [];

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'text-green-500';
      case 'inactive': return 'text-gray-500';
      case 'error': return 'text-red-500';
      case 'connecting': return 'text-yellow-500';
      default: return 'text-gray-500';
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  if (tunnelsLoading || statsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            Tunnel Dashboard
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage and monitor your tunnels
          </p>
        </div>
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
        >
          <Plus size={20} />
          Create Tunnel
        </motion.button>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Total Tunnels</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_tunnels}</p>
              </div>
              <div className="p-3 bg-blue-100 dark:bg-blue-900 rounded-lg">
                <Activity className="w-6 h-6 text-blue-600 dark:text-blue-400" />
              </div>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Active Tunnels</p>
                <p className="text-2xl font-bold text-green-600">{stats.active_tunnels}</p>
              </div>
              <div className="p-3 bg-green-100 dark:bg-green-900 rounded-lg">
                <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
              </div>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Total Bandwidth</p>
                <p className="text-2xl font-bold text-purple-600">{formatBytes(stats.total_bandwidth)}</p>
              </div>
              <div className="p-3 bg-purple-100 dark:bg-purple-900 rounded-lg">
                <HardDrive className="w-6 h-6 text-purple-600 dark:text-purple-400" />
              </div>
            </div>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
            className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Avg Latency</p>
                <p className="text-2xl font-bold text-orange-600">{stats.avg_latency.toFixed(1)}ms</p>
              </div>
              <div className="p-3 bg-orange-100 dark:bg-orange-900 rounded-lg">
                <Clock className="w-6 h-6 text-orange-600 dark:text-orange-400" />
              </div>
            </div>
          </motion.div>
        </div>
      )}

      {/* Filters and Search */}
      <div className="flex flex-col sm:flex-row gap-4 items-center justify-between">
        <div className="flex gap-2">
          {['all', 'active', 'inactive', 'error'].map((status) => (
            <button
              key={status}
              onClick={() => setFilter(status)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                filter === status
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
              }`}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </button>
          ))}
        </div>
        
        <input
          type="text"
          placeholder="Search tunnels..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
      </div>

      {/* Tunnels Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6">
        <AnimatePresence>
          {filteredTunnels.map((tunnel: Tunnel) => (
            <TunnelCard
              key={tunnel.id}
              tunnel={tunnel}
              onAction={handleTunnelAction}
            />
          ))}
        </AnimatePresence>
      </div>

      {filteredTunnels.length === 0 && (
        <div className="text-center py-12">
          <Activity className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
            No tunnels found
          </h3>
          <p className="text-gray-600 dark:text-gray-400 mb-4">
            {searchTerm || filter !== 'all' 
              ? 'Try adjusting your search or filter criteria.'
              : 'Get started by creating your first tunnel.'
            }
          </p>
          {!searchTerm && filter === 'all' && (
            <button
              onClick={() => setShowCreateModal(true)}
              className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
            >
              Create Your First Tunnel
            </button>
          )}
        </div>
      )}

      {/* Modals */}
      <CreateTunnelModal
        isOpen={showCreateModal}
        onClose={() => {
          setShowCreateModal(false);
          setSelectedTunnel(null);
        }}
        tunnel={selectedTunnel}
      />

      <TunnelDetailsModal
        isOpen={showDetailsModal}
        onClose={() => {
          setShowDetailsModal(false);
          setSelectedTunnel(null);
        }}
        tunnel={selectedTunnel}
      />
    </div>
  );
};
