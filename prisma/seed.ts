import { PrismaClient, PostType, SurveyQuestionType, SurveyType, TierListVisibility } from "@prisma/client";

const prisma = new PrismaClient();

async function main() {
  // Users
  const alice = await prisma.user.create({
    data: {
      auth: { create: { provider: "LOCAL", email: "alice@example.com", passwordHash: "dev-only" } },
      profile: { create: { username: "alice", displayName: "Alice" } },
      stats: { create: {} },
      subscriptions: { create: { plan: "PRO", status: "ACTIVE", provider: "dev" } }
    }
  });

  const bob = await prisma.user.create({
    data: {
      auth: { create: { provider: "LOCAL", email: "bob@example.com", passwordHash: "dev-only" } },
      profile: { create: { username: "bob", displayName: "Bob" } },
      stats: { create: {} },
      subscriptions: { create: { plan: "FREE", status: "ACTIVE", provider: "dev" } }
    }
  });

  // Social graph
  await prisma.follow.create({
    data: { followerId: bob.id, followingId: alice.id }
  });

  // Category + master template
  const category = await prisma.category.create({
    data: {
      slug: "coffee",
      name: "Coffee",
      tags: ["drinks", "food"]
    }
  });

  const masterTemplate = await prisma.tierListTemplate.create({
    data: {
      categoryId: category.id,
      isMaster: true,
      title: "Coffee Master Tier List",
      visibility: TierListVisibility.PUBLIC,
      tiers: {
        create: [
          { key: "S", label: "S", order: 1, colorHex: "#ffcc00" },
          { key: "A", label: "A", order: 2, colorHex: "#ffd966" },
          { key: "B", label: "B", order: 3, colorHex: "#c9daf8" },
          { key: "C", label: "C", order: 4, colorHex: "#d9ead3" },
          { key: "D", label: "D", order: 5, colorHex: "#f4cccc" }
        ]
      }
    }
  });

  // Assets
  const latte = await prisma.asset.create({ data: { url: "https://cdn.example.com/dev/latte.jpg", mimeType: "image/jpeg" } });
  const espresso = await prisma.asset.create({ data: { url: "https://cdn.example.com/dev/espresso.jpg", mimeType: "image/jpeg" } });

  // Rank posts
  const post1 = await prisma.post.create({
    data: {
      type: PostType.RANK,
      creatorId: alice.id,
      categoryId: category.id,
      caption: "Latte is S-tier for me",
      rank: {
        create: {
          templateId: masterTemplate.id,
          tierKey: "S",
          imageAssetId: latte.id,
          subjectTitle: "Latte"
        }
      },
      metrics: { create: {} }
    }
  });

  const post2 = await prisma.post.create({
    data: {
      type: PostType.RANK,
      creatorId: bob.id,
      categoryId: category.id,
      caption: "Espresso goes A-tier",
      rank: {
        create: {
          templateId: masterTemplate.id,
          tierKey: "A",
          imageAssetId: espresso.id,
          subjectTitle: "Espresso"
        }
      },
      metrics: { create: {} }
    }
  });

  // Interactions
  await prisma.postLike.create({ data: { postId: post1.id, userId: bob.id } });
  await prisma.comment.create({ data: { postId: post1.id, authorId: bob.id, body: "Hard agree." } });
  await prisma.postShare.create({ data: { postId: post1.id, userId: bob.id, channel: "copy_link" } });

  await prisma.postMetrics.update({
    where: { postId: post1.id },
    data: { likeCount: 1, commentCount: 1, shareCount: 1, hotScore: 4.5 }
  });

  // Pinning
  await prisma.pinnedPost.create({ data: { userId: alice.id, postId: post1.id, order: 1 } });

  // Survey org + survey post + campaign
  const org = await prisma.organization.create({ data: { name: "Acme Research Lab", website: "https://example.com" } });

  const surveyPost = await prisma.post.create({
    data: {
      type: PostType.SURVEY,
      creatorId: alice.id,
      categoryId: category.id,
      caption: "Quick coffee preference survey",
      survey: {
        create: {
          surveyType: SurveyType.THESIS,
          sponsorOrgId: org.id,
          title: "Coffee Habits 2026",
          description: "Anonymous research survey.",
          questions: {
            create: [
              {
                order: 1,
                type: SurveyQuestionType.SINGLE_CHOICE,
                prompt: "How many coffees do you drink per day?",
                options: {
                  create: [
                    { order: 1, label: "0" },
                    { order: 2, label: "1" },
                    { order: 3, label: "2-3" },
                    { order: 4, label: "4+" }
                  ]
                }
              }
            ]
          },
          campaign: {
            create: {
              sponsorOrgId: org.id,
              budgetCents: 5000,
              targetImpressions: 1000,
              targeting: { countries: ["TH", "US"], interests: ["coffee"] }
            }
          }
        }
      },
      metrics: { create: {} }
    }
  });

  // Survey response
  const response = await prisma.surveyResponse.create({
    data: { surveyPostId: surveyPost.id, userId: bob.id }
  });

  const q1 = await prisma.surveyQuestion.findFirstOrThrow({ where: { surveyPostId: surveyPost.id, order: 1 }, include: { options: true } });
  const chosen = q1.options[1];
  if (!chosen) throw new Error("Seed options missing");

  await prisma.surveyAnswer.create({
    data: { responseId: response.id, questionId: q1.id, optionId: chosen.id }
  });

  // Category overall review (illustrative)
  await prisma.categoryOverallReview.create({
    data: { categoryId: category.id, averageTierScore: 4.5, sampleSize: 2, tierDistribution: { S: 1, A: 1 } }
  });

  // Leaderboard snapshot
  const snapshot = await prisma.leaderboardSnapshot.create({
    data: { categoryId: category.id, period: "ALL_TIME" }
  });
  await prisma.leaderboardEntry.createMany({
    data: [
      { snapshotId: snapshot.id, userId: alice.id, score: 10, rank: 1 },
      { snapshotId: snapshot.id, userId: bob.id, score: 7, rank: 2 }
    ]
  });

  // Data market aggregate (anonymized)
  await prisma.dataMarketAggregate.create({
    data: {
      bucketStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000),
      bucketEnd: new Date(),
      categoryId: category.id,
      sampleSize: 2,
      averageTierScore: 4.5,
      tierDistribution: { S: 1, A: 1 }
    }
  });

  // Update user stats (illustrative)
  await prisma.userStats.update({ where: { userId: alice.id }, data: { ranksCreatedCount: 1, followersCount: 1, followingCount: 0 } });
  await prisma.userStats.update({ where: { userId: bob.id }, data: { ranksCreatedCount: 1, followersCount: 0, followingCount: 1 } });

  // Prevent unused variable lints in future tooling
  void post2;
}

main()
  .then(async () => {
    await prisma.$disconnect();
  })
  .catch(async (e) => {
    // eslint-disable-next-line no-console
    console.error(e);
    await prisma.$disconnect();
    process.exit(1);
  });

