<template>
  <section class="hero-section">
    <div class="hero-grid">
      <div>
        <p class="eyebrow">Terminal-native. Geo-capable. Auto-evolving.</p>
        <h1 class="hero-title">Golem turns your Gateway into a vivid, workspace-aware control surface.</h1>
        <p class="hero-copy">
          Operate chat, tools, Geo workflows, and learned execution paths from a product-grade web interface built for people who want both signal and atmosphere.
        </p>
        <div class="hero-actions">
          <RouterLink class="button button-primary" to="/console">Enter Console</RouterLink>
          <a class="button button-ghost" href="#geo-evolution">See Geo + Auto-Evolution</a>
        </div>
      </div>
      <div class="hero-card-stack">
        <article class="signal-card signal-card-primary">
          <span class="signal-label">Gateway</span>
          <strong>health · version · chat</strong>
          <p>One front door into the existing Go backend.</p>
        </article>
        <article class="signal-card">
          <span class="signal-label">Geo</span>
          <strong>GDAL · PostGIS · pipelines</strong>
          <p>Workspace-native spatial execution instead of disconnected tools.</p>
        </article>
        <article class="signal-card signal-card-accent">
          <span class="signal-label">Evolution</span>
          <strong>reuse · scaffold · telemetry</strong>
          <p>Successful workflows leave artifacts the next run can actually use.</p>
        </article>
      </div>
    </div>
  </section>
</template>
