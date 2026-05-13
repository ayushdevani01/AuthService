export type AuthMethods = {
  email: boolean;
  google: boolean;
  github: boolean;
};

export type PublicAppConfig = {
  name: string;
  app_id: string;
  auth_methods: AuthMethods;
  require_email_verification: boolean;
};
