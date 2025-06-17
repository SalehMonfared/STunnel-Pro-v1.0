import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  X, 
  Activity, 
  Clock, 
  Users, 
  Download, 
  Upload, 
  Wifi, 
  WifiOff, 
  Copy, 
  ExternalLink,
  Settings,
  BarChart3,
  AlertTriangle,
  CheckCircle,
  Info
} from 'lucide-react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { toast } from 'react-hot-toast';

interface TunnelDetailsModalProps {
  tunnel: any;
  isOpen: boolean;
  onClose: () => void;
}

export function TunnelDetailsModal({ tunnel, isOpen, onClose }: TunnelDetailsModalProps) {
  const [activeTab, setActiveTab] = useState('overview');
  const [metrics, setMetrics] = useState<any>(null);
  const [logs, setLogs] = useState<any[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  // Mock metrics data
  const metricsData = [
    { time: '00:00', latency: 12, connections: 5, bandwidth: 120 },
    { time: '00:05', latency: 15, connections: 8, bandwidth: 180 },
    { time: '00:10', latency: 11, connections: 12, bandwidth: 250 },
    { time: '00:15', latency: 18, connections: 15, bandwidth: 320 },
    { time: '00:20', latency: 14, connections: 10, bandwidth: 280 },
    { time: '00:25', latency: 16, connections: 7, bandwidth: 200 },
  ];

  useEffect(() => {
    if (isOpen && tunnel) {
      fetchTunnelDetails();
    }
  }, [isOpen, tunnel]);

  const fetchTunnelDetails = async () => {
    setIsLoading(true);
    try {
      const token = localStorage.getItem('access_token');
      
      // Fetch metrics
      const metricsResponse = await fetch(`/api/v1/tunnels/${tunnel.id}/metrics`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      
      if (metricsResponse.ok) {
        const metricsData = await metricsResponse.json();
        setMetrics(metricsData.data);
      }

      // Fetch logs
      const logsResponse = await fetch(`/api/v1/tunnels/${tunnel.id}/logs?limit=50`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      
      if (logsResponse.ok) {
        const logsData = await logsResponse.json();
        setLogs(logsData.data || []);
      }
      
    } catch (error) {
      console.error('Failed to fetch tunnel details:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    toast.success(`${label} copied to clipboard`);
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active': return <CheckCircle className="w-5 h-5 text-green-500" />;
      case 'inactive': return <WifiOff className="w-5 h-5 text-gray-500" />;
      case 'error': return <AlertTriangle className="w-5 h-5 text-red-500" />;
      case 'connecting': return <Activity className="w-5 h-5 text-yellow-500 animate-pulse" />;
      default: return <WifiOff className="w-5 h-5 text-gray-500" />;
    }
  };

  if (!tunnel) return null;

  return (
    <AnimatePresence>
      {isOpen && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
              onClick={onClose}
            />

            {/* Modal */}
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 20 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 20 }}
              className="inline-block w-full max-w-6xl p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white dark:bg-gray-800 shadow-xl rounded-2xl"
            >
              {/* Header */}
              <div className="flex items-center justify-between mb-6">
                <div className="flex items-center space-x-3">
                  {getStatusIcon(tunnel.status)}
                  <div>
                    <h3 className="text-2xl font-bold text-gray-900 dark:text-white">
                      {tunnel.name}
                    </h3>
                    <p className="text-gray-600 dark:text-gray-400">
                      {tunnel.description || 'No description'}
                    </p>
                  </div>
                </div>
                <button
                  onClick={onClose}
                  className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  <X size={24} />
                </button>
              </div>

              {/* Tabs */}
              <div className="flex space-x-1 mb-6 bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
                {[
                  { id: 'overview', label: 'Overview', icon: Info },
                  { id: 'metrics', label: 'Metrics', icon: BarChart3 },
                  { id: 'logs', label: 'Logs', icon: Activity },
                  { id: 'settings', label: 'Settings', icon: Settings },
                ].map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`flex items-center space-x-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                      activeTab === tab.id
                        ? 'bg-white dark:bg-gray-800 text-blue-600 shadow-sm'
                        : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
                    }`}
                  >
                    <tab.icon size={16} />
                    <span>{tab.label}</span>
                  </button>
                ))}
              </div>

              {/* Tab Content */}
              <div className="min-h-[400px]">
                {/* Overview Tab */}
                {activeTab === 'overview' && (
                  <div className="space-y-6">
                    {/* Status Cards */}
                    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                      <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Status</p>
                            <p className={`text-lg font-semibold ${getStatusColor(tunnel.status)}`}>
                              {tunnel.status.charAt(0).toUpperCase() + tunnel.status.slice(1)}
                            </p>
                          </div>
                          {tunnel.is_online ? (
                            <Wifi className="w-8 h-8 text-green-500" />
                          ) : (
                            <WifiOff className="w-8 h-8 text-gray-500" />
                          )}
                        </div>
                      </div>

                      <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Uptime</p>
                            <p className="text-lg font-semibold text-gray-900 dark:text-white">
                              {tunnel.uptime || '0s'}
                            </p>
                          </div>
                          <Clock className="w-8 h-8 text-blue-500" />
                        </div>
                      </div>

                      <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Connections</p>
                            <p className="text-lg font-semibold text-gray-900 dark:text-white">
                              {tunnel.connection_count || 0}
                            </p>
                          </div>
                          <Users className="w-8 h-8 text-purple-500" />
                        </div>
                      </div>

                      <div className="bg-gray-50 dark:bg-gray-700 p-4 rounded-lg">
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Data Transfer</p>
                            <p className="text-lg font-semibold text-gray-900 dark:text-white">
                              {formatBytes((tunnel.bytes_in || 0) + (tunnel.bytes_out || 0))}
                            </p>
                          </div>
                          <Activity className="w-8 h-8 text-orange-500" />
                        </div>
                      </div>
                    </div>

                    {/* Configuration Details */}
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                      {/* Server Configuration */}
                      <div className="bg-gray-50 dark:bg-gray-700 p-6 rounded-lg">
                        <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                          Server Configuration
                        </h4>
                        <div className="space-y-3">
                          <div className="flex justify-between items-center">
                            <span className="text-gray-600 dark:text-gray-400">Protocol:</span>
                            <span className="font-medium text-gray-900 dark:text-white uppercase">
                              {tunnel.protocol}
                            </span>
                          </div>
                          <div className="flex justify-between items-center">
                            <span className="text-gray-600 dark:text-gray-400">Listen Address:</span>
                            <div className="flex items-center space-x-2">
                              <span className="font-medium text-gray-900 dark:text-white">
                                {tunnel.server_ip}:{tunnel.server_port}
                              </span>
                              <button
                                onClick={() => copyToClipboard(`${tunnel.server_ip}:${tunnel.server_port}`, 'Server address')}
                                className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                              >
                                <Copy size={14} />
                              </button>
                            </div>
                          </div>
                          <div className="flex justify-between items-center">
                            <span className="text-gray-600 dark:text-gray-400">Target:</span>
                            <div className="flex items-center space-x-2">
                              <span className="font-medium text-gray-900 dark:text-white">
                                {tunnel.target_ip}:{tunnel.target_port}
                              </span>
                              <button
                                onClick={() => copyToClipboard(`${tunnel.target_ip}:${tunnel.target_port}`, 'Target address')}
                                className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                              >
                                <Copy size={14} />
                              </button>
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* Advanced Settings */}
                      <div className="bg-gray-50 dark:bg-gray-700 p-6 rounded-lg">
                        <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                          Advanced Settings
                        </h4>
                        <div className="space-y-3">
                          <div className="flex justify-between items-center">
                            <span className="text-gray-600 dark:text-gray-400">MUX Enabled:</span>
                            <span className="font-medium text-gray-900 dark:text-white">
                              {tunnel.mux_enabled ? 'Yes' : 'No'}
                            </span>
                          </div>
                          {tunnel.mux_enabled && (
                            <>
                              <div className="flex justify-between items-center">
                                <span className="text-gray-600 dark:text-gray-400">MUX Connections:</span>
                                <span className="font-medium text-gray-900 dark:text-white">
                                  {tunnel.mux_connections || 8}
                                </span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span className="text-gray-600 dark:text-gray-400">Frame Size:</span>
                                <span className="font-medium text-gray-900 dark:text-white">
                                  {formatBytes(tunnel.mux_frame_size || 32768)}
                                </span>
                              </div>
                            </>
                          )}
                          <div className="flex justify-between items-center">
                            <span className="text-gray-600 dark:text-gray-400">TLS Enabled:</span>
                            <span className="font-medium text-gray-900 dark:text-white">
                              {tunnel.tls_enabled ? 'Yes' : 'No'}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Traffic Overview */}
                    <div className="bg-gray-50 dark:bg-gray-700 p-6 rounded-lg">
                      <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                        Traffic Overview
                      </h4>
                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-green-100 dark:bg-green-900 rounded-lg">
                            <Download className="w-5 h-5 text-green-600 dark:text-green-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Bytes In</p>
                            <p className="text-lg font-semibold text-gray-900 dark:text-white">
                              {formatBytes(tunnel.bytes_in || 0)}
                            </p>
                          </div>
                        </div>

                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-blue-100 dark:bg-blue-900 rounded-lg">
                            <Upload className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Bytes Out</p>
                            <p className="text-lg font-semibold text-gray-900 dark:text-white">
                              {formatBytes(tunnel.bytes_out || 0)}
                            </p>
                          </div>
                        </div>

                        <div className="flex items-center space-x-3">
                          <div className="p-2 bg-purple-100 dark:bg-purple-900 rounded-lg">
                            <Activity className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                          </div>
                          <div>
                            <p className="text-sm text-gray-600 dark:text-gray-400">Total</p>
                            <p className="text-lg font-semibold text-gray-900 dark:text-white">
                              {formatBytes((tunnel.bytes_in || 0) + (tunnel.bytes_out || 0))}
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Metrics Tab */}
                {activeTab === 'metrics' && (
                  <div className="space-y-6">
                    <div className="bg-gray-50 dark:bg-gray-700 p-6 rounded-lg">
                      <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                        Performance Metrics
                      </h4>
                      <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={metricsData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="time" />
                          <YAxis />
                          <Tooltip />
                          <Line type="monotone" dataKey="latency" stroke="#3B82F6" strokeWidth={2} name="Latency (ms)" />
                          <Line type="monotone" dataKey="connections" stroke="#10B981" strokeWidth={2} name="Connections" />
                          <Line type="monotone" dataKey="bandwidth" stroke="#8B5CF6" strokeWidth={2} name="Bandwidth (KB/s)" />
                        </LineChart>
                      </ResponsiveContainer>
                    </div>
                  </div>
                )}

                {/* Logs Tab */}
                {activeTab === 'logs' && (
                  <div className="space-y-4">
                    <div className="bg-gray-50 dark:bg-gray-700 p-6 rounded-lg">
                      <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                        Recent Logs
                      </h4>
                      <div className="space-y-2 max-h-96 overflow-y-auto">
                        {logs.length === 0 ? (
                          <p className="text-gray-500 dark:text-gray-400 text-center py-8">
                            No logs available
                          </p>
                        ) : (
                          logs.map((log, index) => (
                            <div key={index} className="flex items-start space-x-3 p-3 bg-white dark:bg-gray-800 rounded-lg">
                              <div className={`w-2 h-2 rounded-full mt-2 ${
                                log.level === 'ERROR' ? 'bg-red-500' :
                                log.level === 'WARN' ? 'bg-yellow-500' :
                                log.level === 'INFO' ? 'bg-blue-500' : 'bg-gray-500'
                              }`} />
                              <div className="flex-1">
                                <div className="flex items-center justify-between">
                                  <span className={`text-xs font-medium ${
                                    log.level === 'ERROR' ? 'text-red-600' :
                                    log.level === 'WARN' ? 'text-yellow-600' :
                                    log.level === 'INFO' ? 'text-blue-600' : 'text-gray-600'
                                  }`}>
                                    {log.level}
                                  </span>
                                  <span className="text-xs text-gray-500 dark:text-gray-400">
                                    {new Date(log.timestamp).toLocaleString()}
                                  </span>
                                </div>
                                <p className="text-sm text-gray-900 dark:text-white mt-1">
                                  {log.message}
                                </p>
                              </div>
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  </div>
                )}

                {/* Settings Tab */}
                {activeTab === 'settings' && (
                  <div className="space-y-6">
                    <div className="bg-gray-50 dark:bg-gray-700 p-6 rounded-lg">
                      <h4 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                        Tunnel Information
                      </h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                        <div>
                          <span className="text-gray-600 dark:text-gray-400">Tunnel ID:</span>
                          <div className="flex items-center space-x-2 mt-1">
                            <span className="font-mono text-gray-900 dark:text-white">{tunnel.id}</span>
                            <button
                              onClick={() => copyToClipboard(tunnel.id, 'Tunnel ID')}
                              className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            >
                              <Copy size={14} />
                            </button>
                          </div>
                        </div>
                        <div>
                          <span className="text-gray-600 dark:text-gray-400">Created:</span>
                          <p className="font-medium text-gray-900 dark:text-white mt-1">
                            {new Date(tunnel.created_at).toLocaleString()}
                          </p>
                        </div>
                        <div>
                          <span className="text-gray-600 dark:text-gray-400">Last Updated:</span>
                          <p className="font-medium text-gray-900 dark:text-white mt-1">
                            {new Date(tunnel.updated_at).toLocaleString()}
                          </p>
                        </div>
                        <div>
                          <span className="text-gray-600 dark:text-gray-400">Last Seen:</span>
                          <p className="font-medium text-gray-900 dark:text-white mt-1">
                            {tunnel.last_seen ? new Date(tunnel.last_seen).toLocaleString() : 'Never'}
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>

              {/* Footer */}
              <div className="flex justify-end space-x-3 pt-6 border-t border-gray-200 dark:border-gray-700">
                <button
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  Close
                </button>
                <button
                  onClick={() => window.open(`http://${tunnel.server_ip}:${tunnel.server_port}`, '_blank')}
                  className="flex items-center space-x-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
                >
                  <ExternalLink size={16} />
                  <span>Open Tunnel</span>
                </button>
              </div>
            </motion.div>
          </div>
        </div>
      )}
    </AnimatePresence>
  );
}
