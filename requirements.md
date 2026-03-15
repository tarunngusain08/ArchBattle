Tarun, given your **PRD v2.2 + Engineering Design Doc**, the fallback plan of **static question seeding** is actually very reasonable and even aligns with your **Stage-1 content pipeline (founder-authored + expert reviewed)**. 

Your system already expects **~80 FFF match questions + Daily Challenge scenarios** for launch, so the right strategy is to **design a structured “question universe” first**, then populate it gradually. 

Below is a **complete content generation plan** tailored to your product mechanics (FFF, Daily Challenge, tiers, discussion threads).

---

# 1. Core Principle: Scenario-Driven System Design

Your questions must not feel like textbook trivia.

Each question should follow this structure:

```
Scenario
↓
System constraints
↓
Architectural decision point
↓
MCQ / multi-select / sequence
↓
Reasoning explanation
```

Example:

```
Scenario
A video streaming platform experiences spikes from 10K → 500K viewers
during live events.

Constraints
• 99.95% availability
• viewers distributed globally
• chat messages must appear <2s delay

Question
Which architecture change should be prioritized first?
```

This aligns with your **“situational specificity” policy** in the PRD. 

---

# 2. Topic Universe (System Design Knowledge Map)

You should organize the entire question bank into **system design domains**.

Think of this as **the ArchBattle knowledge graph**.

---

# A. Traffic & Scalability

Topics:

* Load balancing strategies
* Horizontal vs vertical scaling
* Autoscaling policies
* CDN usage
* Edge computing
* Geo-replication
* Rate limiting

Example scenario ideas:

```
• Live sports streaming traffic spike
• Ticket booking flash sale
• Black Friday ecommerce surge
```

Sample questions:

* Which layer should handle rate limiting?
* When should CDN caching be bypassed?

---

# B. Storage & Databases

Topics:

* SQL vs NoSQL tradeoffs
* Sharding
* Replication
* Consistency models
* Data partitioning
* Hot key mitigation

Scenario examples:

```
• messaging app storing billions of chat messages
• ride-hailing location updates
• payment transaction ledger
```

Question styles:

* best partition key
* replication strategy
* consistency tradeoffs

---

# C. Caching Systems

Topics:

* Redis vs Memcached
* cache invalidation
* cache stampede
* write-through vs write-back
* TTL strategies

Scenario examples:

```
• product catalog cache invalidation
• trending feed updates
• leaderboard queries
```

These work great for **Daily Challenge themes**.

---

# D. Messaging & Event Systems

Topics:

* Kafka vs RabbitMQ
* event sourcing
* exactly-once vs at-least-once
* stream processing
* consumer lag

Scenarios:

```
• ride tracking events
• order processing pipeline
• IoT telemetry ingestion
```

---

# E. Distributed Systems Reliability

Topics:

* circuit breakers
* retries
* backpressure
* cascading failures
* chaos engineering

Scenarios:

```
• payment gateway dependency failing
• retry storm after service outage
• dependency latency spike
```

These are excellent for **Incident Response mode later**.

---

# F. Microservices Architecture

Topics:

* service boundaries
* API gateway
* service discovery
* gRPC vs REST
* service mesh

Scenarios:

```
• migrating monolith to microservices
• versioning breaking API changes
```

---

# G. Real-Time Systems

Topics:

* WebSocket scaling
* pub/sub systems
* real-time leaderboards
* multiplayer synchronization

Scenarios:

```
• real-time multiplayer game
• collaborative document editing
• stock trading dashboard
```

---

# H. Observability & Monitoring

Topics:

* metrics vs logs vs traces
* SLOs
* alerting strategies
* incident detection

Scenarios:

```
• sudden latency spike in microservices
• intermittent API errors
```

---

# I. Security & Multi-Tenant Systems

Topics:

* rate limiting abuse
* API authentication
* tenant isolation
* secrets management

Scenarios:

```
• SaaS platform serving thousands of tenants
• API key abuse
```

---

# J. Data Pipelines & Analytics

Topics:

* batch vs streaming
* ETL pipelines
* warehouse architectures
* data freshness tradeoffs

Scenarios:

```
• analytics pipeline for ecommerce
• real-time fraud detection
```

---

# 3. Question Types (Make the Platform Interesting)

Your PRD allows **various variants**, not only MCQ.

Use these formats:

---

# Type 1 — Classic MCQ

```
Which architecture change improves latency the most?
```

Good for **FFF battles**.

---

# Type 2 — Multi-Select

```
Which TWO changes solve the problem?
```

More complex.

---

# Type 3 — Ordered Decisions

```
What should be done first?
```

Example:

```
A. Increase DB replicas
B. Add caching
C. Scale API servers
D. Introduce sharding
```

---

# Type 4 — Identify the Bottleneck

```
Where is the root cause?
```

Great debugging questions.

---

# Type 5 — Architecture Diagram Completion

```
Which component is missing?
```

Example:

```
Client → API Gateway → ??? → Database
```

---

# Type 6 — Tradeoff Evaluation

```
Why is sharding NOT the best solution here?
```

---

# 4. Launch Content Strategy

Your PRD target:

```
80 FFF questions
14 Daily Challenges (42 questions)
```

Total:

```
122 questions
```

This matches your launch plan. 

---

### Recommended distribution

```
Traffic & scaling      20
Databases              15
Caching                12
Messaging systems      12
Reliability            10
Microservices           6
Realtime systems        7
Security                6
Data pipelines          6
Observability           6
```

---

# 5. Daily Challenge Theme Strategy

Make daily challenges feel **like events**.

Example weekly themes:

### Week 1

```
Caching Week
```

Daily scenarios:

* cache stampede
* read-heavy traffic
* TTL tuning

---

### Week 2

```
Messaging Week
```

Scenarios:

* event ordering
* Kafka partitioning
* consumer lag

---

### Week 3

```
Scaling Week
```

Scenarios:

* autoscaling failures
* load balancer hot spots

---

### Week 4

```
Reliability Week
```

Scenarios:

* retry storms
* cascading failures

---

# 6. Make the Questions “Eye Catchy”

Your questions should read like **engineering war stories**.

Example:

### Instead of

```
What is the best cache strategy?
```

### Use

```
Your ecommerce homepage traffic jumps from
5K RPS → 120K RPS during a flash sale.

Redis CPU spikes to 90%.
Users report slow page loads.

What should you do FIRST?
```

---

# 7. Content Production Workflow

Since you don’t have LLM keys:

### Step 1

Generate **topic + scenario list**

### Step 2

Write:

```
scenario
options
correct answers
rationale
```

### Step 3

Store in Postgres.

Matches your schema:

```
questions
options JSONB
correct_answers
rationale
```

Your design doc already supports this. 

---

# 8. Recommended First 10 Questions

Example ideas:

1️⃣ **CDN vs caching decision**
2️⃣ **Redis hot key meltdown**
3️⃣ **Kafka consumer lag explosion**
4️⃣ **Payment service retry storm**
5️⃣ **Global leaderboard architecture**
6️⃣ **Messaging system ordering guarantee**
7️⃣ **Database shard key selection**
8️⃣ **Autoscaling failure during spike**
9️⃣ **Chat system fan-out architecture**
🔟 **Real-time multiplayer game server scaling**

---

# 9. Long-Term Content Vision

Once discussions accumulate, your platform becomes:

```
Daily challenge
+ community reasoning
+ AI synthesis
```

Exactly the **knowledge commons vision** described in the PRD. 

Over time you get:

```
365 scenarios/year
× discussion threads
× expert insights
```

That becomes **a unique dataset of system design reasoning**.

---

# 10. My Strong Recommendation

Before writing questions, build a **content spreadsheet**:

Columns:

```
topic
scenario
difficulty
question_type
correct_answers
explanation
related_concepts
```

Then write the **first 80 questions offline**.

Only after that populate the database.

