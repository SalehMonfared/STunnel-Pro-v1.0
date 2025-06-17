import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, AlertCircle, Info, Settings, Shield, Zap } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { toast } from 'react-hot-toast';

interface CreateTunnelForm {
  name: string;
  description: string;
  protocol: string;
  server_ip: string;
  server_port: number;
  target_ip: string;
  target_port: number;
  client_ip?: string;
  client_port?: number;
  mux_enabled: boolean;
  mux_connections: number;
  mux_frame_size: number;
  tls_enabled: boolean;
  tls_cert_file?: string;
  tls_key_file?: string;
}

interface CreateTunnelModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  tunnel?: any;
}

export function CreateTunnelModal({ isOpen, onClose, onSuccess, tunnel }: CreateTunnelModalProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');
  
  const {
    register,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<CreateTunnelForm>({
    defaultValues: tunnel ? {
      name: tunnel.name,
      description: tunnel.description,
      protocol: tunnel.protocol,
      server_ip: tunnel.server_ip,
      server_port: tunnel.server_port,
      target_ip: tunnel.target_ip,
      target_port: tunnel.target_port,
      client_ip: tunnel.client_ip,
      client_port: tunnel.client_port,
      mux_enabled: tunnel.mux_enabled ?? true,
      mux_connections: tunnel.mux_connections ?? 8,
      mux_frame_size: tunnel.mux_frame_size ?? 32768,
      tls_enabled: tunnel.tls_enabled ?? false,
      tls_cert_file: tunnel.tls_cert_file,
      tls_key_file: tunnel.tls_key_file,
    } : {
      protocol: 'tcp',
      server_ip: '0.0.0.0',
      mux_enabled: true,
      mux_connections: 8,
      mux_frame_size: 32768,
      tls_enabled: false,
    }
  });

  const protocol = watch('protocol');
  const muxEnabled = watch('mux_enabled');
  const tlsEnabled = watch('tls_enabled');

  const onSubmit = async (data: CreateTunnelForm) => {
    setIsLoading(true);
    
    try {
      const token = localStorage.getItem('access_token');
      const url = tunnel ? `/api/v1/tunnels/${tunnel.id}` : '/api/v1/tunnels';
      const method = tunnel ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify(data),
      });

      const result = await response.json();

      if (response.ok) {
        toast.success(tunnel ? 'Tunnel updated successfully!' : 'Tunnel created successfully!');
        reset();
        onSuccess();
      } else {
        toast.error(result.message || 'Operation failed');
      }
    } catch (error) {
      toast.error('Network error. Please try again.');
      console.error('Tunnel operation error:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const protocolOptions = [
    { value: 'tcp', label: 'TCP', description: 'Standard TCP protocol' },
    { value: 'udp', label: 'UDP', description: 'User Datagram Protocol' },
    { value: 'ws', label: 'WebSocket', description: 'WebSocket protocol' },
    { value: 'wss', label: 'WebSocket Secure', description: 'Secure WebSocket with TLS' },
    { value: 'tcpmux', label: 'TCP Mux', description: 'Multiplexed TCP' },
    { value: 'wsmux', label: 'WS Mux', description: 'Multiplexed WebSocket' },
    { value: 'wssmux', label: 'WSS Mux', description: 'Multiplexed Secure WebSocket' },
  ];

  const getMuxRecommendation = (connections: number) => {
    if (connections <= 4) return { level: 'Basic', color: 'text-blue-600', desc: 'Good for light usage' };
    if (connections <= 16) return { level: 'Optimal', color: 'text-green-600', desc: 'Recommended for most cases' };
    if (connections <= 32) return { level: 'High', color: 'text-orange-600', desc: 'For heavy traffic' };
    return { level: 'Maximum', color: 'text-red-600', desc: 'For extreme loads' };
  };

  const muxConnections = watch('mux_connections') || 8;
  const muxRecommendation = getMuxRecommendation(muxConnections);

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
              className="inline-block w-full max-w-4xl p-6 my-8 overflow-hidden text-left align-middle transition-all transform bg-white dark:bg-gray-800 shadow-xl rounded-2xl"
            >
              {/* Header */}
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-2xl font-bold text-gray-900 dark:text-white">
                  {tunnel ? 'Edit Tunnel' : 'Create New Tunnel'}
                </h3>
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
                  { id: 'basic', label: 'Basic Settings', icon: Settings },
                  { id: 'advanced', label: 'Advanced', icon: Zap },
                  { id: 'security', label: 'Security', icon: Shield },
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

              <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
                {/* Basic Settings Tab */}
                {activeTab === 'basic' && (
                  <div className="space-y-6">
                    {/* Name and Description */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                          Tunnel Name *
                        </label>
                        <input
                          {...register('name', { 
                            required: 'Tunnel name is required',
                            minLength: { value: 3, message: 'Minimum 3 characters required' }
                          })}
                          type="text"
                          className={`block w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                            errors.name 
                              ? 'border-red-300 bg-red-50 dark:bg-red-900/20 dark:border-red-500' 
                              : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700'
                          } text-gray-900 dark:text-white`}
                          placeholder="My Tunnel"
                        />
                        {errors.name && (
                          <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.name.message}</p>
                        )}
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                          Description
                        </label>
                        <input
                          {...register('description')}
                          type="text"
                          className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                          placeholder="Optional description"
                        />
                      </div>
                    </div>

                    {/* Protocol Selection */}
                    <div>
                      <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                        Protocol *
                      </label>
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                        {protocolOptions.map((option) => (
                          <label
                            key={option.value}
                            className={`relative flex flex-col p-3 border rounded-lg cursor-pointer transition-colors ${
                              protocol === option.value
                                ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                                : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
                            }`}
                          >
                            <input
                              {...register('protocol', { required: 'Protocol is required' })}
                              type="radio"
                              value={option.value}
                              className="sr-only"
                            />
                            <span className="font-medium text-gray-900 dark:text-white">{option.label}</span>
                            <span className="text-xs text-gray-500 dark:text-gray-400 mt-1">{option.description}</span>
                          </label>
                        ))}
                      </div>
                      {errors.protocol && (
                        <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.protocol.message}</p>
                      )}
                    </div>

                    {/* Server Configuration */}
                    <div>
                      <h4 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Server Configuration</h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Server IP *
                          </label>
                          <input
                            {...register('server_ip', { 
                              required: 'Server IP is required',
                              pattern: { 
                                value: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$|^0\.0\.0\.0$/,
                                message: 'Invalid IP address'
                              }
                            })}
                            type="text"
                            className={`block w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                              errors.server_ip 
                                ? 'border-red-300 bg-red-50 dark:bg-red-900/20 dark:border-red-500' 
                                : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700'
                            } text-gray-900 dark:text-white`}
                            placeholder="0.0.0.0"
                          />
                          {errors.server_ip && (
                            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.server_ip.message}</p>
                          )}
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Server Port *
                          </label>
                          <input
                            {...register('server_port', { 
                              required: 'Server port is required',
                              min: { value: 1, message: 'Port must be between 1 and 65535' },
                              max: { value: 65535, message: 'Port must be between 1 and 65535' }
                            })}
                            type="number"
                            className={`block w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                              errors.server_port 
                                ? 'border-red-300 bg-red-50 dark:bg-red-900/20 dark:border-red-500' 
                                : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700'
                            } text-gray-900 dark:text-white`}
                            placeholder="8080"
                          />
                          {errors.server_port && (
                            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.server_port.message}</p>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Target Configuration */}
                    <div>
                      <h4 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Target Configuration</h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Target IP *
                          </label>
                          <input
                            {...register('target_ip', { 
                              required: 'Target IP is required',
                              pattern: { 
                                value: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
                                message: 'Invalid IP address'
                              }
                            })}
                            type="text"
                            className={`block w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                              errors.target_ip 
                                ? 'border-red-300 bg-red-50 dark:bg-red-900/20 dark:border-red-500' 
                                : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700'
                            } text-gray-900 dark:text-white`}
                            placeholder="192.168.1.100"
                          />
                          {errors.target_ip && (
                            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.target_ip.message}</p>
                          )}
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Target Port *
                          </label>
                          <input
                            {...register('target_port', { 
                              required: 'Target port is required',
                              min: { value: 1, message: 'Port must be between 1 and 65535' },
                              max: { value: 65535, message: 'Port must be between 1 and 65535' }
                            })}
                            type="number"
                            className={`block w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors ${
                              errors.target_port 
                                ? 'border-red-300 bg-red-50 dark:bg-red-900/20 dark:border-red-500' 
                                : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700'
                            } text-gray-900 dark:text-white`}
                            placeholder="22"
                          />
                          {errors.target_port && (
                            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.target_port.message}</p>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Advanced Settings Tab */}
                {activeTab === 'advanced' && (
                  <div className="space-y-6">
                    {/* Multiplexing Settings */}
                    <div>
                      <div className="flex items-center justify-between mb-4">
                        <h4 className="text-lg font-medium text-gray-900 dark:text-white">Multiplexing (MUX)</h4>
                        <label className="flex items-center">
                          <input
                            {...register('mux_enabled')}
                            type="checkbox"
                            className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                          />
                          <span className="ml-2 text-sm text-gray-700 dark:text-gray-300">Enable MUX</span>
                        </label>
                      </div>

                      {muxEnabled && (
                        <div className="space-y-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              Connections ({muxConnections})
                            </label>
                            <input
                              {...register('mux_connections', { 
                                min: { value: 1, message: 'Minimum 1 connection' },
                                max: { value: 64, message: 'Maximum 64 connections' }
                              })}
                              type="range"
                              min="1"
                              max="64"
                              className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-600"
                            />
                            <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1">
                              <span>1</span>
                              <span className={muxRecommendation.color}>
                                {muxRecommendation.level} - {muxRecommendation.desc}
                              </span>
                              <span>64</span>
                            </div>
                          </div>

                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              Frame Size (bytes)
                            </label>
                            <select
                              {...register('mux_frame_size')}
                              className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                            >
                              <option value={16384}>16 KB (Low latency)</option>
                              <option value={32768}>32 KB (Balanced)</option>
                              <option value={65536}>64 KB (High throughput)</option>
                            </select>
                          </div>
                        </div>
                      )}
                    </div>

                    {/* Client Configuration */}
                    <div>
                      <h4 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Client Configuration (Optional)</h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Client IP
                          </label>
                          <input
                            {...register('client_ip')}
                            type="text"
                            className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                            placeholder="Auto-detect"
                          />
                        </div>

                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                            Client Port
                          </label>
                          <input
                            {...register('client_port')}
                            type="number"
                            className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                            placeholder="Auto-assign"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                {/* Security Settings Tab */}
                {activeTab === 'security' && (
                  <div className="space-y-6">
                    {/* TLS Settings */}
                    <div>
                      <div className="flex items-center justify-between mb-4">
                        <h4 className="text-lg font-medium text-gray-900 dark:text-white">TLS Encryption</h4>
                        <label className="flex items-center">
                          <input
                            {...register('tls_enabled')}
                            type="checkbox"
                            className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                          />
                          <span className="ml-2 text-sm text-gray-700 dark:text-gray-300">Enable TLS</span>
                        </label>
                      </div>

                      {tlsEnabled && (
                        <div className="space-y-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              Certificate File Path
                            </label>
                            <input
                              {...register('tls_cert_file')}
                              type="text"
                              className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                              placeholder="/path/to/certificate.pem"
                            />
                          </div>

                          <div>
                            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                              Private Key File Path
                            </label>
                            <input
                              {...register('tls_key_file')}
                              type="text"
                              className="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                              placeholder="/path/to/private-key.pem"
                            />
                          </div>

                          <div className="flex items-start space-x-2 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                            <Info className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                            <div className="text-sm text-blue-800 dark:text-blue-200">
                              <p className="font-medium mb-1">TLS Certificate Requirements:</p>
                              <ul className="list-disc list-inside space-y-1">
                                <li>Certificate must be in PEM format</li>
                                <li>Private key must match the certificate</li>
                                <li>Files must be accessible by the tunnel process</li>
                              </ul>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {/* Form Actions */}
                <div className="flex justify-end space-x-3 pt-6 border-t border-gray-200 dark:border-gray-700">
                  <button
                    type="button"
                    onClick={onClose}
                    className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={isLoading}
                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {isLoading ? (
                      <div className="flex items-center">
                        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                        {tunnel ? 'Updating...' : 'Creating...'}
                      </div>
                    ) : (
                      tunnel ? 'Update Tunnel' : 'Create Tunnel'
                    )}
                  </button>
                </div>
              </form>
            </motion.div>
          </div>
        </div>
      )}
    </AnimatePresence>
  );
}
