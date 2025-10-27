"use client";

import { useState, useEffect } from "react";
import { MailingList } from "@/lib/cms";
import Icon from "@hackclub/icons";
import { getClosestColor } from "@/lib/utils";

interface EmailFilterProps {
  mailingLists: MailingList[];
  onFilterChange: (filters: {
    includeLists: string[];
    excludeLists: string[];
  }) => void;
}

export function EmailFilter({
  mailingLists,
  onFilterChange,
}: EmailFilterProps) {
  const [includeLists, setIncludeLists] = useState<string[]>([]);
  const [excludeLists, setExcludeLists] = useState<string[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");

  useEffect(() => {
    onFilterChange({ includeLists, excludeLists });
  }, [includeLists, excludeLists, onFilterChange]);

  const filteredMailingLists = mailingLists.filter((list) =>
    list.name.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  const toggleInclude = (slug: string) => {
    setIncludeLists((prev) =>
      prev.includes(slug) ? prev.filter((s) => s !== slug) : [...prev, slug],
    );
    // Remove from exclude if it was there
    setExcludeLists((prev) => prev.filter((s) => s !== slug));
  };

  const toggleExclude = (slug: string) => {
    setExcludeLists((prev) =>
      prev.includes(slug) ? prev.filter((s) => s !== slug) : [...prev, slug],
    );
    // Remove from include if it was there
    setIncludeLists((prev) => prev.filter((s) => s !== slug));
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
          <Icon glyph="filter" className="w-6 h-6 text-muted" />
          <h2 className="text-lg font-semibold text-muted">Filter emails</h2>
          {hasActiveFilters && (
            <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-dark text-blue-800">
              {includeLists.length + excludeLists.length} active
            </span>
          )}
        </div>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="flex items-center space-x-2 px-3 py-2 text-sm font-medium text-primary bg-slate rounded-lg hover:bg-steel transition-colors"
        >
          <span>{isOpen ? "Hide filters" : "Show filters"}</span>
          <svg
            className={`w-4 h-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>
      </div>

      {isOpen && (
        <div className="bg-dark rounded-xl p-6 space-y-6 border border-dark">
          {/* Search */}
          <div>
            <label className="block text-sm font-medium text-snow mb-2">
              Search mailing lists
            </label>
            <div className="relative">
              <Icon
                glyph="search"
                className="w-5 h-5 text-snow absolute left-3 top-1/2 transform -translate-y-1/2"
              />
              <input
                type="text"
                placeholder="Type to search mailing lists..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 text-snow border border-gray-300 rounded-lg focus:ring-2 focus:ring-red focus:border-red transition-colors"
              />
            </div>
          </div>

          {/* Quick Actions */}
          <div>
            <label className="block text-sm font-medium text-primary mb-3">
              Quick actions
            </label>
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => {
                  setIncludeLists(mailingLists.map((l) => l.slug));
                  setExcludeLists([]);
                }}
                className="px-3 py-1 text-sm font-medium text-darkless bg-snow border border-primary rounded-md hover:bg-gray-50 transition-colors"
              >
                Show all
              </button>
              <button
                onClick={() => {
                  setIncludeLists([]);
                  setExcludeLists(mailingLists.map((l) => l.slug));
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
              <Icon glyph="checkmark" className=" text-green h-15" />
              <h3 className="text-sm font-medium text-snow">
                Show only these lists
              </h3>
              {includeLists.length > 0 && (
                <span className="text-xs text-snow">
                  ({includeLists.length} selected)
                </span>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              {filteredMailingLists.map((list) => (
                <button
                  key={list.slug}
                  onClick={() => toggleInclude(list.slug)}
                  className={`px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                    includeLists.includes(list.slug)
                      ? "text-white bg-green shadow-md transform scale-105"
                      : "text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 hover:border-gray-400"
                  }`}
                >
                  <div className="flex items-center space-x-2">
                    <div
                      className={`w-2 h-2 rounded-full ${
                        excludeLists.includes(list.slug) ? "bg-white" : ""
                      }`}
                      style={{ backgroundColor: getClosestColor(list.color) }}
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
              <Icon glyph="checkbox" className=" text-red h-17" />
              <h3 className="text-sm font-medium text-snow">
                Hide these lists
              </h3>
              {excludeLists.length > 0 && (
                <span className="text-xs text-snow">
                  ({excludeLists.length} selected)
                </span>
              )}
            </div>
            <div className="flex flex-wrap gap-2">
              {filteredMailingLists.map((list) => (
                <button
                  key={list.slug}
                  onClick={() => toggleExclude(list.slug)}
                  className={`px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 ${
                    excludeLists.includes(list.slug)
                      ? "text-white bg-red shadow-md transform scale-105"
                      : "text-gray-700 bg-white border border-gray-300 hover:bg-gray-50 hover:border-gray-400"
                  }`}
                >
                  <div className="flex items-center space-x-2">
                    <div
                      className={`w-2 h-2 rounded-full ${
                        excludeLists.includes(list.slug) ? "bg-white" : ""
                      }`}
                      style={{ backgroundColor: getClosestColor(list.color) }}
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
                      <Icon
                        glyph="checkmark"
                        className="
                      text-green h-15"
                      />
                      <span className="text-snow">
                        Showing:{" "}
                        {includeLists
                          .map(
                            (slug) =>
                              mailingLists.find((l) => l.slug === slug)?.name,
                          )
                          .join(", ")}
                      </span>
                    </div>
                  )}
                  {excludeLists.length > 0 && (
                    <div className="flex items-center space-x-2 mt-1">
                      <Icon glyph="checkbox" className=" text-red h-17" />
                      <span className="text-snow">
                        Hiding:{" "}
                        {excludeLists
                          .map(
                            (slug) =>
                              mailingLists.find((l) => l.slug === slug)?.name,
                          )
                          .join(", ")}
                      </span>
                    </div>
                  )}
                </div>
                <button
                  onClick={clearFilters}
                  className="text-sm text-primary bg-red p-2 rounded-lg hover:text-snow font-medium transition-colors"
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
