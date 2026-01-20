import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
    LayoutDashboard,
    Layers,
    Cpu,
    Database,
    Settings,
    Plus,
    Search,
    Activity,
    Zap,
    MessageSquare,
    ChevronRight,
    ExternalLink,
    ShieldCheck,
    Bell,
    Globe,
    Terminal
} from 'lucide-react';

// --- Shared Components ---

interface NavItemProps {
    icon: React.ReactNode;
    label: string;
    active?: boolean;
    onClick?: () => void;
}

const NavItem = ({ icon, label, active = false, onClick }: NavItemProps) => (
    <motion.div
        whileHover={{ x: 5 }}
        whileTap={{ scale: 0.98 }}
        className={`nav-item ${active ? 'active' : ''}`}
        onClick={onClick}
    >
        {icon}
        <span>{label}</span>
    </motion.div>
);

interface StatCardProps {
    icon: React.ReactNode;
    label: string;
    value: string;
    trend: string;
    color: string;
}

const StatCard = ({ icon, label, value, trend, color }: StatCardProps) => (
    <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="glass-panel stat-card"
    >
        <div className="flex justify-between items-start">
            <div className="p-3 rounded-xl bg-white/5 border border-white/5" style={{ color }}>{icon}</div>
            <span className="text-xs font-bold text-accent" style={{ color: trend.startsWith('+') ? '#10b981' : '#ef4444' }}>
                {trend}
            </span>
        </div>
        <div className="stat-value">{value}</div>
        <div className="text-xs text-dim uppercase tracking-wider font-semibold mt-1">{label}</div>
    </motion.div>
);

interface ProjectCardProps {
    name: string;
    status: 'online' | 'building' | 'offline';
    type: string;
    region: string;
    cpu: string;
}

const ProjectCard = ({ name, status, type, region, cpu }: ProjectCardProps) => (
    <motion.div
        whileHover={{ y: -5 }}
        className="glass-panel project-card"
    >
        <div className="flex justify-between items-start">
            <div>
                <h4 className="font-bold text-lg mb-1">{name}</h4>
                <div className={`status-badge ${status === 'online' ? 'status-online' : 'status-building'}`}>
                    <div className={`w-1.5 h-1.5 rounded-full ${status === 'online' ? 'bg-accent' : 'bg-warning'} animate-pulse`} />
                    {status}
                </div>
            </div>
            <button className="p-2 hover:bg-white/5 rounded-lg transition-colors">
                <ExternalLink size={18} className="text-dim hover:text-white" />
            </button>
        </div>

        <div className="grid grid-cols-2 gap-4 pt-4 border-t border-white/5 mt-auto">
            <div className="flex flex-col gap-1">
                <span className="text-[10px] uppercase font-bold text-dim tracking-widest text-secondary">Region</span>
                <span className="text-sm font-medium flex items-center gap-1"><Globe size={12} /> {region}</span>
            </div>
            <div className="flex flex-col gap-1 text-right">
                <span className="text-[10px] uppercase font-bold text-dim tracking-widest text-secondary">Load</span>
                <span className="text-sm font-medium flex items-center justify-end gap-1">{cpu} <Cpu size={12} /></span>
            </div>
        </div>
    </motion.div>
);

// --- Main Application ---

const App: React.FC = () => {
    const [activeTab, setActiveTab] = useState('dashboard');

    return (
        <div className="app-container">
            {/* Sidebar Navigation */}
            <aside className="sidebar">
                <div className="flex items-center justify-center mb-12 px-2">
                    <div className="w-14 h-14 rounded-2xl bg-white/5 flex items-center justify-center overflow-hidden border border-white/10 shadow-2xl">
                        <img
                            src="/logodoric.png"
                            alt="Doric"
                            className="w-[200%] max-w-none -ml-[5%] scale-150"
                            style={{ objectPosition: 'left center' }}
                        />
                    </div>
                </div>

                <nav className="flex flex-col gap-1">
                    <NavItem
                        icon={<LayoutDashboard size={20} />}
                        label="Dashboard"
                        active={activeTab === 'dashboard'}
                        onClick={() => setActiveTab('dashboard')}
                    />
                    <NavItem
                        icon={<Layers size={20} />}
                        label="Deployments"
                        active={activeTab === 'deployments'}
                        onClick={() => setActiveTab('deployments')}
                    />
                    <NavItem
                        icon={<Database size={20} />}
                        label="Databases"
                        active={activeTab === 'databases'}
                        onClick={() => setActiveTab('databases')}
                    />
                    <NavItem
                        icon={<Terminal size={20} />}
                        label="Logs"
                        active={activeTab === 'logs'}
                        onClick={() => setActiveTab('logs')}
                    />
                </nav>

                <div className="mt-auto pt-8 border-t border-white/5">
                    <NavItem icon={<Settings size={20} />} label="Settings" />
                    <NavItem icon={<Bell size={20} />} label="Notifications" />
                </div>
            </aside>

            {/* Main Content Area */}
            <main className="main-content">
                <motion.header
                    initial={{ opacity: 0, y: -20 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="flex justify-between items-center mb-12"
                >
                    <div>
                        <h1 className="text-4xl font-bold mb-2">Cloud Core</h1>
                        <p className="text-secondary font-medium uppercase tracking-[0.2em] text-xs">Node: us-east-1a • Health: Optimal</p>
                    </div>

                    <div className="flex gap-4">
                        <div className="glass-panel flex items-center px-4 py-0 h-12 gap-3 bg-white/5">
                            <Search size={18} className="text-dim" />
                            <input
                                type="text"
                                placeholder="Command center..."
                                className="bg-transparent border-none outline-none text-sm w-48 font-medium"
                            />
                            <span className="text-[10px] bg-white/5 px-1.5 py-0.5 rounded border border-white/10 text-dim">⌘K</span>
                        </div>
                        <button className="btn-primary h-12">
                            <Plus size={20} />
                            <span>Build New</span>
                        </button>
                    </div>
                </motion.header>

                <AnimatePresence mode="wait">
                    {activeTab === 'dashboard' && (
                        <motion.div
                            key="dashboard"
                            initial={{ opacity: 0, x: 20 }}
                            animate={{ opacity: 1, x: 0 }}
                            exit={{ opacity: 0, x: -20 }}
                        >
                            {/* Stats */}
                            <section className="stats-grid">
                                <StatCard icon={<Activity />} label="Global Health" value="99.98%" trend="+0.02%" color="#10b981" />
                                <StatCard icon={<Layers />} label="Containers" value="128" trend="+12" color="#3b82f6" />
                                <StatCard icon={<Cpu />} label="Compute Load" value="42.5%" trend="-4%" color="#8b5cf6" />
                                <StatCard icon={<Zap />} label="Total Requests" value="3.4M" trend="+15%" color="#f59e0b" />
                            </section>

                            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                                {/* Projects List */}
                                <div className="lg:col-span-2 space-y-8">
                                    <div className="flex justify-between items-center">
                                        <h2 className="text-2xl font-bold">Active Projects</h2>
                                        <button className="text-primary text-sm font-bold flex items-center gap-1 hover:brightness-125 transition-all">
                                            Explorer <ChevronRight size={16} />
                                        </button>
                                    </div>

                                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                                        <ProjectCard name="Neural-API-v2" status="online" type="API" region="Paris (FR)" cpu="12%" />
                                        <ProjectCard name="Doric-Front-X" status="building" type="Svelte" region="Dublin (IE)" cpu="84%" />
                                        <ProjectCard name="Auth-Service" status="online" type="Go" region="London (UK)" cpu="5%" />
                                        <ProjectCard name="Media-Transcoder" status="online" type="Node" region="Frankfurt (DE)" cpu="34%" />
                                    </div>
                                </div>

                                {/* Sidebar Stats / AI Insight */}
                                <div className="space-y-8">
                                    <h2 className="text-2xl font-bold">AI Intelligence</h2>
                                    <div className="glass-panel bg-primary/10 border-primary/20 p-6 relative overflow-hidden group">
                                        <div className="relative z-10">
                                            <div className="flex items-center gap-3 mb-4">
                                                <div className="w-10 h-10 rounded-xl bg-primary flex items-center justify-center shadow-lg animate-pulse">
                                                    <Zap size={20} color="white" />
                                                </div>
                                                <span className="font-bold text-sm tracking-wide">Optimization Ready</span>
                                            </div>
                                            <p className="text-sm text-secondary leading-relaxed mb-6">
                                                "Neural-API-v2" is currently underutilized in EU regions. I suggest merging instances to save **$142/mo** without impacting performance.
                                            </p>
                                            <button className="w-full bg-white text-black py-3 rounded-xl font-bold text-sm hover:scale-[1.02] transition-transform">
                                                Execute Optimization
                                            </button>
                                        </div>
                                        <div className="absolute -right-10 -bottom-10 w-40 h-40 bg-primary/20 blur-3xl rounded-full group-hover:bg-primary/30 transition-all" />
                                    </div>

                                    <div className="glass-panel p-6">
                                        <h3 className="font-bold text-sm mb-4 flex items-center gap-2">
                                            <ShieldCheck size={16} className="text-accent" /> Security Pulse
                                        </h3>
                                        <div className="space-y-4">
                                            <div className="flex justify-between items-center">
                                                <span className="text-xs text-dim font-medium">Attack Prevention</span>
                                                <span className="text-xs font-bold text-accent">Active</span>
                                            </div>
                                            <div className="w-full bg-white/5 h-1 rounded-full overflow-hidden">
                                                <motion.div
                                                    initial={{ width: 0 }}
                                                    animate={{ width: '85%' }}
                                                    className="h-full bg-accent"
                                                />
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </motion.div>
                    )}
                </AnimatePresence>
            </main>

            {/* Floating Quick Navigation */}
            <motion.div
                whileHover={{ scale: 1.1 }}
                whileTap={{ scale: 0.9 }}
                className="ai-assistant-bubble"
            >
                <div className="relative">
                    <MessageSquare size={28} color="white" />
                    <div className="absolute -top-1 -right-1 w-3 h-3 bg-accent rounded-full border-2 border-bg-dark" />
                </div>
            </motion.div>
        </div>
    );
};

export default App;
