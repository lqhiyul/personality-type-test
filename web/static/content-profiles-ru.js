(function () {
  // Localized profile bridge: keeps the RU content layer complete when full editorial modules are loaded separately from UI labels.
  const lang = "ru";
  const content = window.APP_CONTENT?.[lang];
  if (!content?.types) return;

  const sectionTitles = {
    image: "Короткий образ",
    logic: "Внутренняя логика",
    thinking: "Как мыслит",
    decisions: "Как принимает решения",
    work: "В работе и учебе",
    communication: "В общении",
    motivation: "Что мотивирует",
    drain: "Что истощает",
    strengths: "Сильные стороны",
    difficulties: "Возможные трудности",
    stress: "Под стрессом",
    growth: "Что помогает развиваться",
    advice: "Практические ориентиры",
    summary: "Короткое резюме",
  };

  function compact(items) {
    return (items || []).filter(Boolean);
  }

  function section(title, paragraphs = [], items = []) {
    return { title, body: compact(paragraphs), items: compact(items) };
  }

  function buildProfile(code, type) {
    const base = window.TYPES_DATA?.[code] || {};
    const summary = type.summary || {};
    const strengths = compact(summary.strengths);
    const difficulties = compact(summary.difficulties);
    const growth = compact(summary.growth);
    const essence = summary.essence || type.tagline || summary.shortSummary || "";
    const shareShort = type.shareText || summary.shortSummary || essence;
    const shareDeep = type.shareDeepText || [shareShort, summary.development].filter(Boolean).join(" ");

    return {
      type: code,
      title: type.name,
      socionics: type.socioCode || base.socioCode || "",
      socionicsLabel: type.socioName || base.socioName || "",
      quadra: type.quadra || base.quadra || "",
      short: {
        essence,
        image: summary.image || essence,
        thinking: summary.thinkingStyle || essence,
        strengths,
        difficulties,
        growth,
        workStyle: summary.workStyle || essence,
        communicationStyle: summary.communicationStyle || essence,
        development: summary.development || growth.join(" "),
        summary: summary.shortSummary || essence,
      },
      full: [
        section(sectionTitles.image, [summary.image || essence]),
        section(sectionTitles.logic, [summary.thinkingStyle || essence]),
        section(sectionTitles.thinking, [summary.thinkingStyle || essence, type.tagline || ""]),
        section(sectionTitles.decisions, [essence, summary.development || ""]),
        section(sectionTitles.work, [summary.workStyle || essence]),
        section(sectionTitles.communication, [summary.communicationStyle || essence]),
        section(sectionTitles.motivation, [essence], strengths.slice(0, 3)),
        section(sectionTitles.drain, [], difficulties.slice(0, 3)),
        section(sectionTitles.strengths, [], strengths),
        section(sectionTitles.difficulties, [], difficulties),
        section(sectionTitles.stress, [difficulties.slice(0, 2).join(" ") || essence]),
        section(sectionTitles.growth, [], growth),
        section(sectionTitles.advice, [summary.development || growth.join(" ")]),
        section(sectionTitles.summary, [summary.shortSummary || essence]),
      ],
      share: { short: shareShort, deep: shareDeep },
    };
  }

  function toAppType(profile) {
    return {
      profile,
      name: profile.title,
      socioCode: profile.socionics,
      socioName: profile.socionicsLabel,
      quadra: profile.quadra,
      tagline: profile.short.essence,
      shareText: profile.share.short,
      shareDeepText: profile.share.deep,
      summary: {
        essence: profile.short.essence,
        image: profile.short.image,
        thinkingStyle: profile.short.thinking,
        strengths: profile.short.strengths,
        difficulties: profile.short.difficulties,
        growth: profile.short.growth,
        workStyle: profile.short.workStyle,
        communicationStyle: profile.short.communicationStyle,
        development: profile.short.development,
        shortSummary: profile.short.summary,
      },
      sections: [],
      fullProfile: {
        title: profile.title,
        lead: profile.short.essence,
        sections: profile.full.map((item) => ({
          title: item.title,
          paragraphs: item.body || [],
          items: item.items || [],
        })),
      },
      source: false,
    };
  }

  const profiles = {};
  Object.entries(content.types).forEach(([code, type]) => {
    const profile = buildProfile(code, type);
    profiles[code] = profile;
    content.types[code] = { ...type, ...toAppType(profile) };
  });

  window.APP_PROFILE_CONTENT = window.APP_PROFILE_CONTENT || {};
  window.APP_PROFILE_CONTENT[lang] = profiles;
})();
