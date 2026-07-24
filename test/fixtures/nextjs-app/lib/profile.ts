export interface Profile {
  name: string;
  email: string;
  avatar: string;
}

export async function getProfile(): Promise<Profile> {
  return {
    name: 'Test User',
    email: 'user@example.com',
    avatar: '/avatar.png',
  };
}
