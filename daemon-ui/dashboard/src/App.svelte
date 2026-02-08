<script>
  import Overview from './pages/Overview.svelte';
  import Providers from './pages/Providers.svelte';
  import Apps from './pages/Apps.svelte';
  import History from './pages/History.svelte';
  import Approve from './pages/Approve.svelte';
  import Onboarding from './pages/Onboarding.svelte';
  import { getOnboardingStatus } from './lib/api.js';

  let hash = $state(window.location.hash || '#/');
  let onboardingLoading = $state(true);
  let setupComplete = $state(false);

  $effect(() => {
    getOnboardingStatus()
      .then(data => {
        setupComplete = data.setup_complete;
      })
      .catch(() => {
        // If the endpoint fails, assume setup is complete (graceful degradation)
        setupComplete = true;
      })
      .finally(() => {
        onboardingLoading = false;
      });
  });

  $effect(() => {
    function onHashChange() {
      hash = window.location.hash || '#/';
    }
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  });

  let route = $derived(hash.split('?')[0]);

  const navItems = [
    { href: '#/', label: 'Overview', icon: '\u25A3' },
    { href: '#/providers', label: 'Providers', icon: '\u2699' },
    { href: '#/apps', label: 'Apps', icon: '\u25CB' },
    { href: '#/history', label: 'History', icon: '\u2630' },
    { href: '#/approve', label: 'Approve', icon: '\u2713' },
  ];

  function isActive(href) {
    if (href === '#/') return route === '#/' || route === '#' || route === '';
    return route.startsWith(href);
  }

  function handleOnboardingComplete() {
    setupComplete = true;
  }
</script>

{#if onboardingLoading}
  <div class="loading">Loading...</div>
{:else if !setupComplete}
  <Onboarding onComplete={handleOnboardingComplete} />
{:else}
  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-brand">
        <h1>plug-my-ai</h1>
        <div class="subtitle">local AI proxy</div>
      </div>
      <nav class="sidebar-nav">
        {#each navItems as item}
          <a href={item.href} class:active={isActive(item.href)}>
            <span class="nav-icon">{item.icon}</span>
            {item.label}
          </a>
        {/each}
      </nav>
      <div class="sidebar-footer">
        localhost:21110
      </div>
    </aside>

    <main class="main">
      {#if route === '#/' || route === '#' || route === ''}
        <Overview />
      {:else if route === '#/providers'}
        <Providers />
      {:else if route === '#/apps'}
        <Apps />
      {:else if route === '#/history'}
        <History />
      {:else if route.startsWith('#/approve')}
        <Approve {hash} />
      {:else}
        <div class="empty-state">
          <div class="icon">?</div>
          <h3>Page not found</h3>
          <p>The page <code>{route}</code> does not exist.</p>
        </div>
      {/if}
    </main>
  </div>
{/if}
