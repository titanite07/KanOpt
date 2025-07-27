import { useState, useEffect, useRef, useCallback } from 'react';

export interface VirtualScrollConfig {
  itemHeight: number;
  itemCount: number;
  overscan?: number;
  containerHeight?: number;
  threshold?: number;
}

export interface VirtualScrollReturn {
  containerRef: React.RefObject<HTMLDivElement>;
  visibleRange: { start: number; end: number };
  scrollToItem: (index: number) => void;
  isItemVisible: (index: number) => boolean;
  totalHeight: number;
}

export const useVirtualScroll = (config: VirtualScrollConfig): VirtualScrollReturn => {
  const {
    itemHeight,
    itemCount,
    overscan = 3,
    containerHeight = 600,
    threshold = 100,
  } = config;

  const containerRef = useRef<HTMLDivElement>(null);
  const [scrollTop, setScrollTop] = useState(0);
  const [isScrolling, setIsScrolling] = useState(false);
  
  const scrollTimeoutRef = useRef<NodeJS.Timeout>();

  const totalHeight = itemCount * itemHeight;
  const visibleCount = Math.ceil(containerHeight / itemHeight);

  // Calculate visible range with overscan
  const visibleRange = {
    start: Math.max(0, Math.floor(scrollTop / itemHeight) - overscan),
    end: Math.min(
      itemCount - 1,
      Math.floor(scrollTop / itemHeight) + visibleCount + overscan
    ),
  };

  // Optimized scroll handler with throttling
  const handleScroll = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;

    const newScrollTop = container.scrollTop;
    
    // Only update if scroll position changed significantly
    if (Math.abs(newScrollTop - scrollTop) > threshold) {
      setScrollTop(newScrollTop);
    }

    setIsScrolling(true);

    // Clear existing timeout
    if (scrollTimeoutRef.current) {
      clearTimeout(scrollTimeoutRef.current);
    }

    // Set scrolling to false after scroll stops
    scrollTimeoutRef.current = setTimeout(() => {
      setIsScrolling(false);
    }, 150);
  }, [scrollTop, threshold]);

  // Debounced scroll handler using RAF
  const rafRef = useRef<number>();
  const rafScrollHandler = useCallback(() => {
    if (rafRef.current) {
      cancelAnimationFrame(rafRef.current);
    }
    
    rafRef.current = requestAnimationFrame(() => {
      handleScroll();
    });
  }, [handleScroll]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    container.addEventListener('scroll', rafScrollHandler, { passive: true });

    return () => {
      container.removeEventListener('scroll', rafScrollHandler);
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }
    };
  }, [rafScrollHandler]);

  // Scroll to specific item
  const scrollToItem = useCallback((index: number) => {
    const container = containerRef.current;
    if (!container) return;

    const targetScrollTop = index * itemHeight;
    container.scrollTo({
      top: targetScrollTop,
      behavior: 'smooth',
    });
  }, [itemHeight]);

  // Check if item is visible
  const isItemVisible = useCallback((index: number) => {
    return index >= visibleRange.start && index <= visibleRange.end;
  }, [visibleRange]);

  // Intersection Observer for better performance
  useEffect(() => {
    const container = containerRef.current;
    if (!container || !('IntersectionObserver' in window)) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          const target = entry.target as HTMLElement;
          const index = parseInt(target.dataset.index || '0', 10);
          
          if (entry.isIntersecting) {
            // Item is visible
            target.style.opacity = '1';
            target.style.transform = 'translateY(0)';
          } else {
            // Item is not visible - could implement recycling here
            target.style.opacity = '0.8';
            target.style.transform = 'translateY(5px)';
          }
        });
      },
      {
        root: container,
        rootMargin: `${threshold}px`,
        threshold: [0, 0.25, 0.5, 0.75, 1],
      }
    );

    // Observe all visible items
    const items = container.querySelectorAll('[data-index]');
    items.forEach((item) => observer.observe(item));

    return () => {
      observer.disconnect();
    };
  }, [threshold, visibleRange]);

  return {
    containerRef,
    visibleRange,
    scrollToItem,
    isItemVisible,
    totalHeight,
  };
};

// Hook for detecting scroll direction
export const useScrollDirection = () => {
  const [scrollDirection, setScrollDirection] = useState<'up' | 'down' | null>(null);
  const [lastScrollY, setLastScrollY] = useState(0);

  useEffect(() => {
    const updateScrollDirection = () => {
      const scrollY = window.pageYOffset;
      
      if (Math.abs(scrollY - lastScrollY) < 10) {
        return;
      }
      
      setScrollDirection(scrollY > lastScrollY ? 'down' : 'up');
      setLastScrollY(scrollY > 0 ? scrollY : 0);
    };

    const throttledUpdateScrollDirection = throttle(updateScrollDirection, 100);
    
    window.addEventListener('scroll', throttledUpdateScrollDirection);
    
    return () => {
      window.removeEventListener('scroll', throttledUpdateScrollDirection);
    };
  }, [lastScrollY]);

  return scrollDirection;
};

// Utility throttle function
function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle: boolean;
  return function (this: any, ...args: Parameters<T>) {
    if (!inThrottle) {
      func.apply(this, args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}
