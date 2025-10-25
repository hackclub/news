'use client';

import { useState, useEffect } from 'react';
import { MailingList } from '@/lib/cms';

interface EmailFilterProps {
  mailingLists: MailingList[];
  onFilterChange: (filters: {
    includeLists: string[];
    excludeLists: string[];
  }) => void;
}

export function EmailFilter({ mailingLists, onFilterChange }: EmailFilterProps) {
  const [includeLists, setIncludeLists] = useState<string[]>([]);
  const [excludeLists, setExcludeLists] = useState<string[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    onFilterChange({ includeLists, excludeLists });
  }, [includeLists, excludeLists, onFilterChange]);

  const filteredMailingLists = mailingLists.filter(list =>
    list.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const toggleInclude = (slug: string) => {
    setIncludeLists(prev => 
      prev.includes(slug) 
        ? prev.filter(s => s !== slug)
        : [...prev, slug]
    );
    // Remove from exclude if it was there
    setExcludeLists(prev => prev.filter(s => s !== slug));
  };

  const toggleExclude = (slug: string) => {
    setExcludeLists(prev => 
      prev.includes(slug) 
        ? prev.filter(s => s !== slug)
        : [...prev, slug]
    );
    // Remove from include if it was there
    setIncludeLists(prev => prev.filter(s => s !== slug));
  };

  const clearFilters = () => {
    setIncludeLists([]);
    setExcludeLists([]);
  };

  const hasActiveFilters = includeLists.length > 0 || excludeLists.length > 0;

  return (
    <div className="mb-8">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <svg className="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
          </svg>
          <h2 className="text-lg font-semibold text-gray-900">Filter emails</h2>
          {hasActiveFilters && (
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
              {includeLists.length + excludeLists.length} active
            </span>
          )}
        </div>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="flex items-center space-x-2 px-3 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
        >
          <span>{isOpen ? 'Hide filters' : 'Show filters'}</span>
          <svg 
            className={`w-4 h-4 transition-transform ${isOpen ? 'rotate-180' : ''}`} 
            fill="none" 
            stroke="currentColor" 
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </div>

      {isOpen && (
        <div className="bg-gray-50 rounded-xl p-6 space-y-6 border border-gray-200">
          {/* Search */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Search mailing lists
            </label>
            <div className="relative">
              <svg className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                type="text"
                placeholder="Type to search mailing lists..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
              />
            </div>
          </div>

          {/* Quick Actions */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Quick actions
            </label>
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => {
                  setIncludeLists(mailingLists.map(l => l.slug));
                  setExcludeLists([]);
                }}
                className="px-3 py-1 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
              >
                Show all
              </button>
              <button
                onClick={() => {
                  setIncludeLists([]);
                  setExcludeLists(mailingLists.map(l => l.slug));
                }}
                className="px-3 py-1 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
              >
                Hide all
              </button>
              <button
                onClick={clearFilters}
                className="px-3 py-1 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
              >
                Clear filters
              </button>
            </div>
          </div>

          {/* Include Lists */}
          <div>
            <div className="flex items-center space-x-2 mb-3">
              <svg className="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              <h3 className="text-sm font-medium text-gray-900">Show only these lists</h3>
              {includeLists.length > 0 && (
                <span className="text-xs text-gray-500">({includeLists.length} selected)</span>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              {filteredMailingLists.map((list) => (
                <button
                  key={list.slug}
                  onClick={() => toggleInclude(list.slug)}
                  className={`px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                    includeLists.includes(list.slug)
                      ? 'text-white shadow-md transform scale-105'
                      : 'text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 hover:border-gray-400'
                  }`}
                  style={includeLists.includes(list.slug) ? { backgroundColor: list.color } : {}}
                >
                  <div className="flex items-center space-x-2">
                    <div 
                      className={`w-2 h-2 rounded-full ${
                        includeLists.includes(list.slug) ? 'bg-white' : ''
                      }`}
                      style={includeLists.includes(list.slug) ? {} : { backgroundColor: list.color }}
                    />
                    <span>{list.name}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Exclude Lists */}
          <div>
            <div className="flex items-center space-x-2 mb-3">
              <svg className="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
              <h3 className="text-sm font-medium text-gray-900">Hide these lists</h3>
              {excludeLists.length > 0 && (
                <span className="text-xs text-gray-500">({excludeLists.length} selected)</span>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              {filteredMailingLists.map((list) => (
                <button
                  key={list.slug}
                  onClick={() => toggleExclude(list.slug)}
                  className={`px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                    excludeLists.includes(list.slug)
                      ? 'text-white bg-red-500 shadow-md transform scale-105'
                      : 'text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 hover:border-gray-400'
                  }`}
                >
                  <div className="flex items-center space-x-2">
                    <div 
                      className={`w-2 h-2 rounded-full ${
                        excludeLists.includes(list.slug) ? 'bg-white' : ''
                      }`}
                      style={excludeLists.includes(list.slug) ? {} : { backgroundColor: list.color }}
                    />
                    <span>{list.name}</span>
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Active Filters Summary */}
          {hasActiveFilters && (
            <div className="pt-4 border-t border-gray-200">
              <div className="flex items-center justify-between">
                <div className="text-sm text-gray-600">
                  {includeLists.length > 0 && (
                    <div className="flex items-center space-x-2">
                      <svg className="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                      <span>
                        Showing: {includeLists.map(slug => 
                          mailingLists.find(l => l.slug === slug)?.name
                        ).join(', ')}
                      </span>
                    </div>
                  )}
                  {excludeLists.length > 0 && (
                    <div className="flex items-center space-x-2 mt-1">
                      <svg className="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                      <span>
                        Hiding: {excludeLists.map(slug => 
                          mailingLists.find(l => l.slug === slug)?.name
                        ).join(', ')}
                      </span>
                    </div>
                  )}
                </div>
                <button
                  onClick={clearFilters}
                  className="text-sm text-red-600 hover:text-red-800 font-medium transition-colors"
                >
                  Clear all
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
