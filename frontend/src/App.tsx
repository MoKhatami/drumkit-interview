import React, { useState, useEffect } from 'react';
import './App.css';

interface Load {
    id: string;
    origin: string;
    destination: string;
    customer: string;
    carrier: string;
    status: string;
    created_at: string;
}

interface FormData {
    customer: string;
    pickup: string;
    pickupState: string;
    pickupCountry: string;
    delivery: string;
    deliveryState: string;
    deliveryCountry: string;
}

const App: React.FC = () => {
    const [loads, setLoads] = useState<Load[]>([]);
    const [loading, setLoading] = useState<boolean>(false);
    const [showForm, setShowForm] = useState<boolean>(false);
    const [toast, setToast] = useState<string>('');
    const [maxLoads, setMaxLoads] = useState<number>(10);
    const [formData, setFormData] = useState<FormData>({
        customer: '',
        pickup: '',
        pickupState: '',
        pickupCountry: '',
        delivery: '',
        deliveryState: '',
        deliveryCountry: ''
    });

    const fetchLoads = async (): Promise<void> => {
        setLoading(true);
        try {
            const response = await fetch(`/api/loads`);
            const data = await response.json();
            setLoads(data || []);
        } catch (error) {
            console.error('Error fetching loads:', error);
        }
        setLoading(false);
    };

    const handleSubmit = async (e: React.FormEvent): Promise<void> => {
        e.preventDefault();
        setLoading(true);

        try {
            const loadData = {
                origin: `${formData.pickup}, ${formData.pickupState}`,
                destination: `${formData.delivery}, ${formData.deliveryState}`,
                customer: formData.customer,
                carrier: 'Default Carrier', // You might want to add this to the form
                status: 'active'
            };

            const response = await fetch(`/api/loads`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(loadData),
            });

            if (response.ok) {
                setToast('Load created successfully!');
                setShowForm(false);
                setFormData({
                    customer: '',
                    pickup: '',
                    pickupState: '',
                    pickupCountry: '',
                    delivery: '',
                    deliveryState: '',
                    deliveryCountry: ''
                });
                
                fetchLoads();
            } else {
                setToast('Failed to create load');
            }
        } catch (error) {
            console.error('Error creating load:', error);
            setToast('Error creating load');
        }
        setLoading(false);
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
        const { name, value } = e.target;
        setFormData(prev => ({
            ...prev,
            [name]: value
        }));
    };

    const deleteLoad = async (id: string) => {
        await fetch(`/api/loads?id=${id}`, { method: 'DELETE' });
        fetchLoads();
    };

    useEffect(() => {
        fetchLoads();
    }, []);

    useEffect(() => {
        if (toast) {
            const timer = setTimeout(() => setToast(''), 3000);
            return () => clearTimeout(timer);
        }
    }, [toast]);

    return (
        <div className="App">
            <header style={{ padding: '20px', backgroundColor: '#f5f5f5', borderBottom: '1px solid #ddd' }}>
                <h1>TMS Load Management</h1>
                <p>Drumkit x Turvo Integration</p>
            </header>

            <main style={{ padding: '20px', maxWidth: '1200px', margin: '0 auto' }}>
                {toast && (
                    <div style={{
                        padding: '10px',
                        marginBottom: '20px',
                        backgroundColor: '#d4edda',
                        color: '#155724',
                        border: '1px solid #c3e6cb',
                        borderRadius: '4px'
                    }}>
                        {toast}
                    </div>
                )}

                <div style={{ marginBottom: '20px' }}>
                    <button
                        onClick={() => setShowForm(!showForm)}
                        style={{
                            padding: '10px 20px',
                            backgroundColor: '#007bff',
                            color: 'white',
                            border: 'none',
                            borderRadius: '4px',
                            cursor: 'pointer',
                            marginRight: '10px'
                        }}
                    >
                        {showForm ? 'Cancel' : 'Create New Load'}
                    </button>
                    
                    <button
                        onClick={fetchLoads}
                        disabled={loading}
                        style={{
                            padding: '10px 20px',
                            backgroundColor: '#28a745',
                            color: 'white',
                            border: 'none',
                            borderRadius: '4px',
                            cursor: loading ? 'not-allowed' : 'pointer',
                            opacity: loading ? 0.6 : 1,
                            marginRight: '10px'
                        }}
                    >
                        {loading ? 'Loading...' : 'Refresh'}
                    </button>
                    
                    <label>
                        Show: 
                        <select 
                            value={maxLoads} 
                            onChange={(e) => setMaxLoads(Number(e.target.value))}
                            style={{ marginLeft: '5px', padding: '5px' }}
                        >
                            <option value={5}>5 loads</option>
                            <option value={10}>10 loads</option>
                            <option value={15}>15 loads</option>
                            <option value={25}>25 loads</option>
                            <option value={50}>50 loads</option>
                        </select>
                    </label>
                </div>

                {showForm && (
                    <div style={{
                        backgroundColor: '#f8f9fa',
                        padding: '20px',
                        borderRadius: '8px',
                        marginBottom: '20px',
                        border: '1px solid #dee2e6'
                    }}>
                        <h2>Create New Load</h2>
                        <form onSubmit={handleSubmit}>
                            {/* Customer Section */}
                            <div style={{ marginBottom: '20px' }}>
                                <div>
                                    <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                        Customer Name:
                                    </label>
                                    <input
                                        type="text"
                                        name="customer"
                                        value={formData.customer}
                                        onChange={handleInputChange}
                                        required
                                        style={{
                                            width: '100%',
                                            padding: '8px',
                                            border: '1px solid #ccc',
                                            borderRadius: '4px'
                                        }}
                                    />
                                </div>
                            </div>

                            {/* Pickup Section */}
                            <div style={{ marginBottom: '20px' }}>
                                <h3 style={{ marginBottom: '10px', color: '#495057' }}>Pickup Location</h3>
                                <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: '15px' }}>
                                    <div>
                                        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                            City:
                                        </label>
                                        <input
                                            type="text"
                                            name="pickup"
                                            value={formData.pickup}
                                            onChange={handleInputChange}
                                            required
                                            style={{
                                                width: '100%',
                                                padding: '8px',
                                                border: '1px solid #ccc',
                                                borderRadius: '4px'
                                            }}
                                        />
                                    </div>

                                    <div>
                                        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                            State:
                                        </label>
                                        <input
                                            type="text"
                                            name="pickupState"
                                            value={formData.pickupState}
                                            onChange={handleInputChange}
                                            required
                                            style={{
                                                width: '100%',
                                                padding: '8px',
                                                border: '1px solid #ccc',
                                                borderRadius: '4px'
                                            }}
                                        />
                                    </div>

                                    <div>
                                        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                            Country:
                                        </label>
                                        <input
                                            type="text"
                                            name="pickupCountry"
                                            value={formData.pickupCountry}
                                            onChange={handleInputChange}
                                            required
                                            style={{
                                                width: '100%',
                                                padding: '8px',
                                                border: '1px solid #ccc',
                                                borderRadius: '4px'
                                            }}
                                        />
                                    </div>
                                </div>
                            </div>

                            {/* Delivery Section */}
                            <div style={{ marginBottom: '20px' }}>
                                <h3 style={{ marginBottom: '10px', color: '#495057' }}>Delivery Location</h3>
                                <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr', gap: '15px' }}>
                                    <div>
                                        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                            City:
                                        </label>
                                        <input
                                            type="text"
                                            name="delivery"
                                            value={formData.delivery}
                                            onChange={handleInputChange}
                                            required
                                            style={{
                                                width: '100%',
                                                padding: '8px',
                                                border: '1px solid #ccc',
                                                borderRadius: '4px'
                                            }}
                                        />
                                    </div>

                                    <div>
                                        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                            State:
                                        </label>
                                        <input
                                            type="text"
                                            name="deliveryState"
                                            value={formData.deliveryState}
                                            onChange={handleInputChange}
                                            required
                                            style={{
                                                width: '100%',
                                                padding: '8px',
                                                border: '1px solid #ccc',
                                                borderRadius: '4px'
                                            }}
                                        />
                                    </div>

                                    <div>
                                        <label style={{ display: 'block', marginBottom: '5px', fontWeight: 'bold' }}>
                                            Country:
                                        </label>
                                        <input
                                            type="text"
                                            name="deliveryCountry"
                                            value={formData.deliveryCountry}
                                            onChange={handleInputChange}
                                            required
                                            style={{
                                                width: '100%',
                                                padding: '8px',
                                                border: '1px solid #ccc',
                                                borderRadius: '4px'
                                            }}
                                        />
                                    </div>
                                </div>
                            </div>

                            <button
                                type="submit"
                                disabled={loading}
                                style={{
                                    padding: '10px 20px',
                                    backgroundColor: '#007bff',
                                    color: 'white',
                                    border: 'none',
                                    borderRadius: '4px',
                                    cursor: loading ? 'not-allowed' : 'pointer',
                                    opacity: loading ? 0.6 : 1
                                }}
                            >
                                {loading ? 'Creating...' : 'Create Load'}
                            </button>
                        </form>
                    </div>
                )}

                <div>
                    <h2>Loads ({loads.length} total)</h2>
                    {loading && !showForm ? (
                        <div style={{ textAlign: 'center', padding: '20px' }}>
                            <div>Loading loads...</div>
                        </div>
                    ) : (
                        <div style={{ overflowX: 'auto' }}>
                            <table style={{
                                width: '100%',
                                borderCollapse: 'collapse',
                                backgroundColor: 'white',
                                boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
                            }}>
                                <thead>
                                    <tr style={{ backgroundColor: '#f8f9fa' }}>
                                        <th style={{ padding: '12px', textAlign: 'left', borderBottom: '2px solid #dee2e6' }}>Load ID</th>
                                        <th style={{ padding: '12px', textAlign: 'left', borderBottom: '2px solid #dee2e6' }}>Customer</th>
                                        <th style={{ padding: '12px', textAlign: 'left', borderBottom: '2px solid #dee2e6' }}>Carrier</th>
                                        <th style={{ padding: '12px', textAlign: 'left', borderBottom: '2px solid #dee2e6' }}>Route</th>
                                        <th style={{ padding: '12px', textAlign: 'left', borderBottom: '2px solid #dee2e6' }}>Status</th>
                                        <th style={{ padding: '12px', textAlign: 'left', borderBottom: '2px solid #dee2e6' }}>Actions</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {loads.length === 0 ? (
                                        <tr>
                                            <td colSpan={6} style={{ padding: '20px', textAlign: 'center', color: '#666' }}>
                                                No loads found
                                            </td>
                                        </tr>
                                    ) : (
                                        loads.slice(0, maxLoads).map((load, index) => (
                                            <tr key={load.id || index} style={{ borderBottom: '1px solid #dee2e6' }}>
                                                <td style={{ padding: '12px' }}>{load.id}</td>
                                                <td style={{ padding: '12px' }}>{load.customer || 'Unknown'}</td>
                                                <td style={{ padding: '12px' }}>{load.carrier || 'Unknown'}</td>
                                                <td style={{ padding: '12px' }}>{load.origin} → {load.destination}</td>
                                                <td style={{ padding: '12px' }}>
                                                    <span style={{
                                                        padding: '4px 8px',
                                                        borderRadius: '4px',
                                                        backgroundColor: load.status === 'active' ? '#d4edda' : '#f8d7da',
                                                        color: load.status === 'active' ? '#155724' : '#721c24',
                                                        fontSize: '12px'
                                                    }}>
                                                        {load.status}
                                                    </span>
                                                </td>
                                                <td style={{ padding: '12px' }}>
                                                    <button
                                                        onClick={() => deleteLoad(load.id)}
                                                        style={{
                                                            padding: '4px 8px',
                                                            backgroundColor: '#dc3545',
                                                            color: 'white',
                                                            border: 'none',
                                                            borderRadius: '4px',
                                                            cursor: 'pointer',
                                                            fontSize: '12px'
                                                        }}
                                                    >
                                                        Delete
                                                    </button>
                                                </td>
                                            </tr>
                                        ))
                                    )}
                                </tbody>
                            </table>
                        </div>
                    )}
                </div>
            </main>
        </div>
    );
};

export default App;
