const DRAFT_KEY = "personality-test-draft:v3";
const LANG_KEY = "personality-test-language";
const TELEGRAM_URL = "https://t.me/+H1RfT8lJFYA0MDI6";
const LANGS = ["uk", "ru", "en"];
const TYPE_GRID_ORDER = ["INTJ", "INTP", "ENTJ", "ENTP", "INFJ", "INFP", "ENFJ", "ENFP", "ISTJ", "ISFJ", "ESTJ", "ESFJ", "ISTP", "ISFP", "ESTP", "ESFP"];
const COMPATIBILITY_CONTEXTS = ["friendship", "relationship", "work"];
const SHARE_CARD_ASSETS = Object.freeze(Object.fromEntries(TYPE_GRID_ORDER.map((type) => [type, `/assets/share-cards/${type.toLowerCase()}.png`])));
const QUESTION_METADATA = [
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["energy", "social-recharge"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["group-entry", "communication"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["thinking-process"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["recovery"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["teamwork", "pace"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["outer-style"] },
  { axis: "EI", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["idea-sharing"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["perception", "evidence"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["information-style"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["ideas", "application"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["attention"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["learning-style"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["planning"] },
  { axis: "SN", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["abstraction"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["decision-making"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["feedback"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["conflict"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["impact"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["mistakes"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["communication-tone"] },
  { axis: "TF", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["conflict-regulation"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["planning", "task-load"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["deadlines"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["starting-work"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["rest"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["everyday-rhythm"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["change"] },
  { axis: "JP", function: null, socionicsAspect: null, weight: 1, reverse: false, tags: ["work-style"] },
];

const CONTENT = window.APP_CONTENT || {};
const state = {
  answers: [],
  adminResults: [],
  lastResult: null,
  lang: "uk",
  startedAt: Date.now(),
  typeFilter: "all",
  typeSearch: "",
  activeType: "",
  adminCardVisible: false,
  adminAccessOpen: false,
  authPanelOpen: false,
  authMode: "login",
  currentUser: null,
  profileResults: [],
  profileLoading: false,
  friends: [],
  friendRequests: [],
  friendsLoading: false,
  friendRequestsLoading: false,
  publicProfile: null,
  publicProfileLoading: false,
  publicProfileError: "",
  publicProfileUsername: "",
  profileEditOpen: false,
  profileEditAvatarKey: "gradient-violet",
  demoRunId: 0,
  demoRunning: false,
  compatibility: {
    typeA: "",
    typeB: "",
    context: "friendship",
    result: null,
  },
};

