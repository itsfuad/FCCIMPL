// Homepage scroll animations and mouse effects

// Intersection Observer for scroll animations
const observerOptions = {
  threshold: 0.1,
  rootMargin: '0px 0px -50px 0px'
};

const observer = new IntersectionObserver((entries) => {
  entries.forEach(entry => {
    if (entry.isIntersecting) {
      entry.target.classList.add('visible');
      // Optional: Unobserve after animation (remove to animate on every scroll)
      // observer.unobserve(entry.target);
    }
  });
}, observerOptions);

// Observe all elements with scroll animation classes
function initScrollAnimations() {
  const animatedElements = document.querySelectorAll('.scroll-fade-in, .scroll-slide-up');
  animatedElements.forEach(el => observer.observe(el));
}

// Mouse position tracking for interactive background
function initMouseTracking() {
  const modernLayout = document.querySelector('.modern-layout');
  
  if (!modernLayout) return;
  
  let rafId = null;
  let mouseX = 50;
  let mouseY = 50;
  
  modernLayout.addEventListener('mousemove', (e) => {
    if (rafId) {
      cancelAnimationFrame(rafId);
    }
    
    rafId = requestAnimationFrame(() => {
      const rect = modernLayout.getBoundingClientRect();
      mouseX = ((e.clientX - rect.left) / rect.width) * 100;
      mouseY = ((e.clientY - rect.top) / rect.height) * 100;
      
      modernLayout.style.setProperty('--mouse-x', `${mouseX}%`);
      modernLayout.style.setProperty('--mouse-y', `${mouseY}%`);
    });
  });
  
  // Reset position when mouse leaves
  modernLayout.addEventListener('mouseleave', () => {
    if (rafId) {
      cancelAnimationFrame(rafId);
    }
    modernLayout.style.setProperty('--mouse-x', '50%');
    modernLayout.style.setProperty('--mouse-y', '50%');
  });
}

// Subtle parallax effect on scroll for hero section
function initParallax() {
  const hero = document.querySelector('.hero');
  if (!hero) return;
  
  let ticking = false;
  
  window.addEventListener('scroll', () => {
    if (!ticking) {
      window.requestAnimationFrame(() => {
        const scrolled = window.pageYOffset;
        const rate = scrolled * 0.3;
        
        if (scrolled < window.innerHeight) {
          hero.style.transform = `translateY(${rate}px)`;
          hero.style.opacity = `${1 - (scrolled / window.innerHeight) * 0.5}`;
        }
        
        ticking = false;
      });
      
      ticking = true;
    }
  });
}

// Add magnetic effect to action buttons
function initMagneticButtons() {
  const buttons = document.querySelectorAll('.hero .action, .path-link');
  
  buttons.forEach(button => {
    button.addEventListener('mousemove', (e) => {
      const rect = button.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      
      const centerX = rect.width / 2;
      const centerY = rect.height / 2;
      
      const deltaX = (x - centerX) / centerX;
      const deltaY = (y - centerY) / centerY;
      
      const moveX = deltaX * 5;
      const moveY = deltaY * 5;
      
      button.style.transform = `translate(${moveX}px, ${moveY}px)`;
    });
    
    button.addEventListener('mouseleave', () => {
      button.style.transform = '';
    });
  });
}

// Add smooth cursor trail effect
function initCursorTrail() {
  const modernLayout = document.querySelector('.modern-layout');
  if (!modernLayout) return;
  
  // Create cursor trail container
  const trail = document.createElement('div');
  trail.className = 'cursor-trail';
  trail.style.cssText = `
    position: fixed;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: radial-gradient(circle, rgba(244, 69, 151, 0.4), transparent);
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.3s ease;
    z-index: 9999;
    mix-blend-mode: screen;
  `;
  document.body.appendChild(trail);
  
  let mouseX = 0;
  let mouseY = 0;
  let trailX = 0;
  let trailY = 0;
  
  modernLayout.addEventListener('mousemove', (e) => {
    mouseX = e.clientX;
    mouseY = e.clientY;
    trail.style.opacity = '1';
  });
  
  modernLayout.addEventListener('mouseleave', () => {
    trail.style.opacity = '0';
  });
  
  // Smooth trail following
  function animateTrail() {
    trailX += (mouseX - trailX) * 0.15;
    trailY += (mouseY - trailY) * 0.15;
    
    trail.style.left = `${trailX - 10}px`;
    trail.style.top = `${trailY - 10}px`;
    
    requestAnimationFrame(animateTrail);
  }
  
  animateTrail();
}

// Add ripple effect on click
function initRippleEffect() {
  const clickableElements = document.querySelectorAll('.path-link, .hero .action, .feature-item');
  
  clickableElements.forEach(element => {
    element.addEventListener('click', function(e) {
      const ripple = document.createElement('span');
      const rect = this.getBoundingClientRect();
      
      const size = Math.max(rect.width, rect.height);
      const x = e.clientX - rect.left - size / 2;
      const y = e.clientY - rect.top - size / 2;
      
      ripple.style.cssText = `
        position: absolute;
        width: ${size}px;
        height: ${size}px;
        border-radius: 50%;
        background: rgba(244, 69, 151, 0.3);
        left: ${x}px;
        top: ${y}px;
        pointer-events: none;
        animation: ripple-animation 0.6s ease-out;
      `;
      
      this.style.position = 'relative';
      this.style.overflow = 'hidden';
      this.appendChild(ripple);
      
      setTimeout(() => ripple.remove(), 600);
    });
  });
  
  // Add ripple animation to document if not exists
  if (!document.getElementById('ripple-styles')) {
    const style = document.createElement('style');
    style.id = 'ripple-styles';
    style.textContent = `
      @keyframes ripple-animation {
        from {
          transform: scale(0);
          opacity: 1;
        }
        to {
          transform: scale(2);
          opacity: 0;
        }
      }
    `;
    document.head.appendChild(style);
  }
}

// Initialize on DOM ready
function init() {
  initScrollAnimations();
  initMouseTracking();
  initParallax();
  initMagneticButtons();
  initCursorTrail();
  initRippleEffect();
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

// Re-initialize on View Transitions navigation (Astro)
document.addEventListener('astro:page-load', init);
document.addEventListener('astro:after-swap', init);
