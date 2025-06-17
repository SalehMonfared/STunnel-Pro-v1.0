import { useEffect, useState } from 'react';
import { useRouter } from 'next/router';
import { motion } from 'framer-motion';
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
  HardDrive,
  Wifi,
  WifiOff,
  MoreVertical,
  Edit,
  Copy,
  ExternalLink
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area, PieChart, Pie, Cell } from 'recharts';
import { toast } from 'react-hot-toast';

interface User {
  id: string;
  username: string;
  email: string;
  first_name: string;
  last_name: string;
  role: string;
}

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

export default function DashboardPage() {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [tunnels, setTunnels] = useState<Tunnel[]>([]);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedTunnel, setSelectedTunnel] = useState<Tunnel | null>(null);
  const [filter, setFilter] = useState<string>('all');

  // Mock data for charts
  const trafficData = [
    { time: '00:00', in: 120, out: 80 },
    { time: '04:00', in: 150, out: 100 },
    { time: '08:00', in: 300, out: 200 },
    { time: '12:00', in: 450, out: 350 },
    { time: '16:00', in: 380, out: 280 },
    { time: '20:00', in: 250, out: 180 },
  ];

  const protocolData = [
    { name: 'TCP', value: 45, color: '#3B82F6' },
    { name: 'UDP', value: 25, color: '#10B981' },
    { name: 'WSS', value: 20, color: '#8B5CF6' },
    { name: 'WS', value: 10, color: '#F59E0B' },
  ];

  useEffect(() => {
    checkAuth();
    fetchDashboardData();
    
    // Set up real-time updates
    const interval = setInterval(fetchDashboardData, 30000);
    return () => clearInterval(interval);
  }, []);

  const checkAuth = () => {
    const token = localStorage.getItem('access_token');
    const userData = localStorage.getItem('user');
    
    if (!token || !userData) {
      router.push('/login');
      return;
    }
    
    setUser(JSON.parse(userData));
  };

  const fetchDashboardData = async () => {
    try {
      const token = localStorage.getItem('access_token');
      
      // Fetch tunnels
      const tunnelsResponse = await fetch('/api/v1/tunnels', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
      
      if (tunnelsResponse.ok) {
        const tunnelsData = await tunnelsResponse.json();
        setTunnels(tunnelsData.data || []);
      }

      // Fetch stats
      const statsResponse = await fetch('/api/v1/dashboard/stats', {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      });
      
      if (statsResponse.ok) {
        const statsData = await statsResponse.json();
        setStats(statsData.data);
      }
      
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
      toast.error('Failed to load dashboard data');
    } finally {
      setIsLoading(false);
    }
  };

  const handleTunnelAction = async (action: string, tunnel: Tunnel) => {
    const token = localStorage.getItem('access_token');
    
    try {
      let response;
      
      switch (action) {
        case 'start':
          response = await fetch(`/api/v1/tunnels/${tunnel.id}/start`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}` },
          });
          break;
        case 'stop':
          response = await fetch(`/api/v1/tunnels/${tunnel.id}/stop`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}` },
          });
          break;
        case 'delete':
          if (!confirm('Are you sure you want to delete this tunnel?')) return;
          response = await fetch(`/api/v1/tunnels/${tunnel.id}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` },
          });
          break;
        default:
          return;
      }

      if (response?.ok) {
        toast.success(`Tunnel ${action}ed successfully`);
        fetchDashboardData();
      } else {
        const error = await response?.json();
        toast.error(error.message || `Failed to ${action} tunnel`);
      }
    } catch (error) {
      toast.error(`Failed to ${action} tunnel`);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'text-green-500';
      case 'inactive': return 'text-gray-500';
      case 'error': return 'text-red-500';
      case 'connecting': return 'text-yellow-500';
      default: return 'text-gray-500';
    }
  };

  const getStatusBg = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'inactive': return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
      case 'error': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      case 'connecting': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      default: return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200';
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const filteredTunnels = tunnels.filter(tunnel => {
    if (filter === 'all') return true;
    return tunnel.status === filter;
  });

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <header className="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <Activity className="w-8 h-8 text-blue-600 mr-3" />
              <h1 className="text-xl font-semibold text-gray-900 dark:text-white">
                STunnel Pro v1.0 Dashboard
              </h1>
            </div>
            
            <div className="flex items-center space-x-4">
              <button
                onClick={() => setShowCreateModal(true)}
                className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center gap-2 transition-colors"
              >
                <Plus size={20} />
                Create Tunnel
              </button>
              
              <div className="flex items-center space-x-2">
                <img
                  className="h-8 w-8 rounded-full"
                  src={`https://ui-avatars.com/api/?name=${user?.first_name}+${user?.last_name}&background=3B82F6&color=fff`}
                  alt={user?.username}
                />
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {user?.first_name} {user?.last_name}
                </span>
              </div>
            </div>
          </div>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700"
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600 dark:text-gray-400">Total Tunnels</p>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">{stats?.total_tunnels || 0}</p>
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
                <p className="text-2xl font-bold text-green-600">{stats?.active_tunnels || 0}</p>
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
                <p className="text-2xl font-bold text-purple-600">{formatBytes(stats?.total_bandwidth || 0)}</p>
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
                <p className="text-2xl font-bold text-orange-600">{stats?.avg_latency?.toFixed(1) || 0}ms</p>
              </div>
              <div className="p-3 bg-orange-100 dark:bg-orange-900 rounded-lg">
                <Clock className="w-6 h-6 text-orange-600 dark:text-orange-400" />
              </div>
            </div>
          </motion.div>
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
          {/* Traffic Chart */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Traffic Overview</h3>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={trafficData}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Area type="monotone" dataKey="in" stackId="1" stroke="#3B82F6" fill="#3B82F6" fillOpacity={0.6} />
                <Area type="monotone" dataKey="out" stackId="1" stroke="#10B981" fill="#10B981" fillOpacity={0.6} />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Protocol Distribution */}
          <div className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Protocol Distribution</h3>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={protocolData}
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  dataKey="value"
                  label={({ name, value }) => `${name}: ${value}%`}
                >
                  {protocolData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Tunnels List */}
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <div className="flex justify-between items-center">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Tunnels</h3>
              
              {/* Filter Buttons */}
              <div className="flex gap-2">
                {['all', 'active', 'inactive', 'error'].map((status) => (
                  <button
                    key={status}
                    onClick={() => setFilter(status)}
                    className={`px-3 py-1 rounded-lg text-sm font-medium transition-colors ${
                      filter === status
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                    }`}
                  >
                    {status.charAt(0).toUpperCase() + status.slice(1)}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="p-6">
            {filteredTunnels.length === 0 ? (
              <div className="text-center py-12">
                <Activity className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  No tunnels found
                </h3>
                <p className="text-gray-600 dark:text-gray-400 mb-4">
                  Get started by creating your first tunnel.
                </p>
                <button
                  onClick={() => setShowCreateModal(true)}
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
                >
                  Create Your First Tunnel
                </button>
              </div>
            ) : (
              <div className="grid gap-4">
                {filteredTunnels.map((tunnel) => (
                  <motion.div
                    key={tunnel.id}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 hover:shadow-md transition-shadow"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-4">
                        <div className={`p-2 rounded-lg ${tunnel.is_online ? 'bg-green-100 dark:bg-green-900' : 'bg-gray-100 dark:bg-gray-700'}`}>
                          {tunnel.is_online ? (
                            <Wifi className="w-5 h-5 text-green-600 dark:text-green-400" />
                          ) : (
                            <WifiOff className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                          )}
                        </div>
                        
                        <div>
                          <h4 className="font-medium text-gray-900 dark:text-white">{tunnel.name}</h4>
                          <p className="text-sm text-gray-600 dark:text-gray-400">{tunnel.description}</p>
                          <div className="flex items-center space-x-4 mt-1">
                            <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusBg(tunnel.status)}`}>
                              {tunnel.status}
                            </span>
                            <span className="text-xs text-gray-500 dark:text-gray-400">
                              {tunnel.protocol.toUpperCase()}
                            </span>
                            <span className="text-xs text-gray-500 dark:text-gray-400">
                              {tunnel.server_ip}:{tunnel.server_port} → {tunnel.target_ip}:{tunnel.target_port}
                            </span>
                          </div>
                        </div>
                      </div>

                      <div className="flex items-center space-x-2">
                        {tunnel.status === 'active' ? (
                          <button
                            onClick={() => handleTunnelAction('stop', tunnel)}
                            className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900 rounded-lg transition-colors"
                            title="Stop tunnel"
                          >
                            <Pause size={16} />
                          </button>
                        ) : (
                          <button
                            onClick={() => handleTunnelAction('start', tunnel)}
                            className="p-2 text-green-600 hover:bg-green-50 dark:hover:bg-green-900 rounded-lg transition-colors"
                            title="Start tunnel"
                          >
                            <Play size={16} />
                          </button>
                        )}
                        
                        <button
                          onClick={() => setSelectedTunnel(tunnel)}
                          className="p-2 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900 rounded-lg transition-colors"
                          title="View details"
                        >
                          <Eye size={16} />
                        </button>
                        
                        <button
                          onClick={() => handleTunnelAction('delete', tunnel)}
                          className="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900 rounded-lg transition-colors"
                          title="Delete tunnel"
                        >
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </div>

                    {tunnel.is_online && (
                      <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">Uptime:</span>
                          <span className="ml-2 font-medium text-gray-900 dark:text-white">{tunnel.uptime}</span>
                        </div>
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">Connections:</span>
                          <span className="ml-2 font-medium text-gray-900 dark:text-white">{tunnel.connection_count}</span>
                        </div>
                        <div>
                          <span className="text-gray-500 dark:text-gray-400">Data:</span>
                          <span className="ml-2 font-medium text-gray-900 dark:text-white">
                            ↓{formatBytes(tunnel.bytes_in)} ↑{formatBytes(tunnel.bytes_out)}
                          </span>
                        </div>
                      </div>
                    )}
                  </motion.div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Create Tunnel Modal */}
      {showCreateModal && (
        <CreateTunnelModal
          isOpen={showCreateModal}
          onClose={() => setShowCreateModal(false)}
          onSuccess={() => {
            setShowCreateModal(false);
            fetchDashboardData();
          }}
        />
      )}

      {/* Tunnel Details Modal */}
      {selectedTunnel && (
        <TunnelDetailsModal
          tunnel={selectedTunnel}
          isOpen={!!selectedTunnel}
          onClose={() => setSelectedTunnel(null)}
        />
      )}
    </div>
  );
}

// Placeholder components - will be implemented next
function CreateTunnelModal({ isOpen, onClose, onSuccess }: any) {
  return null; // Will implement next
}

function TunnelDetailsModal({ tunnel, isOpen, onClose }: any) {
  return null; // Will implement next
}
